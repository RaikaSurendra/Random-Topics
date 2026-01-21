(function process( /*RESTAPIRequest*/ request, body) {
    //gs.debug('Inside em azure processor, body: ' + body + ', headers: ' + JSON.stringify(request.headers));
    try {
        var requestBody = JSON.parse(body);


        // hardcoded check for 'AlertSender' role, can change this as needed
        var authHeader = request.getHeader("Authorization");
        var roleToCheck = gs.getProperty("sn_em_connector.AzureMonitorAlertSenderAADRole");

        //gs.info("Azure Monitor Debug: authHeader is " + authHeader + " and role check is " + roleToCheck + " and current user is" + gs.getUserName());
        //gs.info("Azure Monitor Debug:   request body is " + JSON.stringify(requestBody));

        if (!isRequiredRolePresent(authHeader, roleToCheck)) {
            var unauthorizedError = new sn_ws_err.ServiceError();
            unauthorizedError.setStatus(401);
            unauthorizedError.setMessage("Unauthenticated request");
            unauthorizedError.setDetail("Required role claim is not present in token");
            response.setError(unauthorizedError);
            return;
        }

        //Check whether its common schema or not
        var schemaId = requestBody.schemaId;
        if (schemaId != "azureMonitorCommonAlertSchema") {
            gs.error("AzureMonitor Push Connector: Azure Monitor supports only Common Schema. SchemaId is not azureMonitorCommonAlertSchema. This Event will not be created.");
            gs.debug("SchemaId is not azureMonitorCommonAlertSchema. Payload received is " + JSON.stringify(requestBody));
            var NotCommonSchemaError = new sn_ws_err.ServiceError();
            NotCommonSchemaError.setStatus(400);
            NotCommonSchemaError.setMessage("Azure Monitor supports only Common Schema.");
            NotCommonSchemaError.setDetail("Azure Monitor supports only Common Schema.");
            response.setError(NotCommonSchemaError);
            return;
        }
		//flattened fields has to be seperated and added to additional information.
		var flattenPayload = new sn_em_connector.FlattenPayload();
		var additional_info = flattenPayload.getFlattenPayloadWithFirstLevel(body); //FirstLevel flattened properties also added to additional_info
		
		// implement resource here
        var payload = requestBody;
        var webhookData = requestBody.data;
        var name = webhookData.essentials.alertRule;
        var signalType = webhookData.essentials.signalType;
        var monitorService = webhookData.essentials.monitoringService;
        var alertTargetIDs = webhookData.essentials.alertTargetIDs;
        var alertSeverity = webhookData.essentials.severity;
        var monitorCondition = webhookData.essentials.monitorCondition;
        var firedDateTime = webhookData.essentials.firedDateTime;
        var description = webhookData.essentials.description;
        schemaId = payload.schemaId;
        var id = webhookData.essentials.alertId;
        var subscriptionId = "";
        var resourceType = "";
        var providersStartIndex = 0;
        var resourceName = "";
        var subscriptionStartIndex = 0;
        var subscriptionEndIndex = 0;
        var messageKeyAppend = "";
		var monitorType = "";
		var bindToName = "";


        //var sysid_name = [];

        var event_gr = new GlideRecord("em_event");
        event_gr.initialize();
		
		//update additional_info field field with all the query_params from the endpoint
		var endpointParamsUtil = new EndpointParamsUtil();
		endpointParamsUtil.updateAdditionalInfoWithEndpointParams(request.queryParams, additional_info);

        event_gr.source = source_label;
        //In Event Rules - Source - Category to which this matching rule applies. The mapping rule only applies to events with the same event class value. If this value is empty, apply the rule to all events.
        //don't change this event_class value.
        event_gr.event_class = source_label;


        if (!monitorService.startsWith("Activity Log")) {
            event_gr.type = signalType + "-" + monitorService;
        } else {
            event_gr.type = monitorService;
        }

        //changes for stry51622078
		//Adding name to messagekey for log alerts STRY51685355
        if ( monitorService == "Platform" ) {
            messageKeyAppend = webhookData.alertContext.condition.allOf[0].metricName;
        }
        else if ( monitorService == "Application Insights" ) {
            messageKeyAppend = webhookData.alertContext.ApplicationId + name;
			monitorType = "logAlerts";
        }
        else if ( monitorService == "Log Analytics" ) {
            messageKeyAppend = webhookData.alertContext.AffectedConfigurationItems[0] + name;
			monitorType = "logAlerts";
        }
        else if( monitorService == "Log Alerts V2" ) {
            messageKeyAppend = name;  //removing searchquery as it may contain special characters that may cause issue.
			monitorType = "logAlerts";
        }
        else if ( monitorService.startsWith("Activity Log") ||  monitorService == "ServiceHealth" ||  monitorService == "Resource Health" ) {
           messageKeyAppend = webhookData.alertContext.operationName;
        }
        else {
            messageKeyAppend = monitorService + "__UnknownMonitoringServiceToSNow";
        }

        if( messageKeyAppend === undefined) {
            messageKeyAppend = "UndefinedAttributeValue";
        }

        event_gr.metric_name = name;
        event_gr.time_of_event = new GlideDateTime(firedDateTime.substr(0, firedDateTime.indexOf('.')).replace('T', ' '));


        if (description == null || description == "null" || description.trim() == "") {
            description = "The " + signalType + " rule " + name + " with the severity " + alertSeverity + " was " + monitorCondition;
            event_gr.description = description;
        } else {
            event_gr.description = description;
        }


        if (monitorCondition == "Fired") {
            event_gr.resolution_state = "New";

            //changes for stry51622078
            //  Sev0 from Azure to Critical-1 on ServiceNow side
            //  Sev1 from Azure to Major-2 on ServiceNow side
            //  Sev2, sev3  from Azure to Warning on ServiceNow side.
			//  STRY51644997 change sev4 to ok on servicenow side

            var numSeverity = Number(alertSeverity.substr(-1));
            if ( numSeverity == 0 || numSeverity == 1 || numSeverity == 4) {
                event_gr.severity = ( numSeverity + 1).toString();
            }
            else {
                //mapping to warning severity
                event_gr.severity = "4";
            }
        } else {
            event_gr.resolution_state = "Closing";
            event_gr.severity = "0"; //Severity is set to CLEAR as its closing event
        }

        var responseBody = {};
        responseBody.eventSysIds = [];


        //If there are multiple targetIDs, creating one event for each targetID
        alertTargetIDs.forEach(function(targetID) {

            subscriptionStartIndex = targetID.indexOf("/subscriptions/") + 15;
            subscriptionEndIndex = targetID.indexOf("/", subscriptionStartIndex);

            if (subscriptionEndIndex == -1) {
                subscriptionId = targetID.substring(subscriptionStartIndex);
            } else {
                subscriptionId = targetID.substring(subscriptionStartIndex, subscriptionEndIndex);
            }

            providersStartIndex = targetID.indexOf("/providers/") + 11;
             
			// /subscriptions/xyz/resourceGroups/movedfromsm-Naga/providers/Microsoft.Compute/virtualMachines/maheshazuredev
			//resourceType will be Microsoft.Compute/virtualMachines and resourceName will be maheshazuredev
            if (providersStartIndex == 10 ) { //10 as targetID.indexOf("/providers/")  will give -1 and we are adding 11.
                resourceType = "";
                resourceName = "";
            } else {
				//STRY51692972 - parse resourceType and resourceName correctly
                var resourceArray =  targetID.substring(providersStartIndex).split("/");
                resourceType = resourceArray[0]; 
				
                for(var index = 1; index < resourceArray.length; index++ ) {
                  if(index % 2 == 1) {
                    resourceType = resourceType + "/" + resourceArray[index];
                  } else {
                     resourceName = resourceName + "/" + resourceArray[index];
                  } 
                }
            }
			if(resourceName != "") {
				resourceName = resourceName.substring(1);
			}
			
			//stry STRY51692972: to bind to the dimension reosurce ratherthan alerttarget
			if(monitorType == "logAlerts") {
				var dimensions = webhookData.alertContext.Dimensions;
				if(dimensions === undefined) {
					monitorType = "";
				} else {
					for ( index = 0; index < dimensions.length;index++) {
						if (dimensions[index].Name == 'Computer' ||  dimensions[index].Name == 'Resource' ||  dimensions[index].Name == '_ResourceId' ||  dimensions[index].Name ==  'ResourceId' ) {
							bindToName = dimensions[index].Value;
							break;
						}
					}
				}

			}
			
			//stry STRY51692972: to bind to the dimension reosurce ratherthan alerttarget
			if(monitorType == "logAlerts" && bindToName != "") { // here we want bind to the resource in dimension as the alertTargetID will be pointing to workspace.
				additional_info["name"] = bindToName;
				additional_info["monitorType"] = monitorType;
				additional_info["object_id_1"] = targetID; //intentionally keeping it as object_id_1 so that binding  happens with name
			}
			else {
				additional_info["object_id"] = targetID;
			}
            
            additional_info["azureData"] = payload;
            additional_info["subscriptionId"] = subscriptionId;
            additional_info["resourceType"] = resourceType;
            additional_info["resourceName"] = resourceName;
            additional_info["schemaId"] = schemaId;

            event_gr.additional_info = JSON.stringify(additional_info);


            event_gr.message_key =  targetID + "__" + monitorService + "__"  + bindToName + messageKeyAppend ; //event_gr.cmdb_ci;
            responseBody.eventSysIds.push(event_gr.insert());
        });


        // can additionally get sys id of alert associated via event rule
        response.setBody(responseBody);
    } catch (er) {
        gs.info(er);
        status = 500;
        return er;
    }
    return "success";
})(request, body);

//// check if role assigned to AMP app is present in token for verification,
//// first argument is the request auth header, second  is the role to verify.
function isRequiredRolePresent(authHeader, roleToCheck) {
    //gs.info("Azure Monitor Debug: Entered isRequiredRolePresent method");

    if (authHeader == null || !roleToCheck) {
        // using some other authentication header or not configured role to check as sys property
        return true;
    }

    var spaceIndex = authHeader.indexOf(' ');
    if (spaceIndex < 0 || authHeader.substr(0, spaceIndex).toLowerCase() != "bearer") {
        // not bearer token
        return true;
    }

    // if bearer token, assume its AAD, then validate role claim
    var accessToken = authHeader.substr(spaceIndex + 1);

    var firstDot = accessToken.indexOf('.'),
        lastDot = accessToken.lastIndexOf('.');
    var encodedClaims = accessToken.substr(firstDot + 1, lastDot - firstDot - 1);

    var claims = gs.base64Decode(encodedClaims);
    //gs.info("Azure Monitor Debug: Claims==== " + claims);
    if (typeof claims === "undefined") {
        claims = GlideStringUtil.base64Decode(encodedClaims);
    }
    var claimsObj = JSON.parse(claims);
    var roleClaims = claimsObj.roles;

    //multiple values has to be given, give comma seperated values.
    var roleToCheckArray = roleToCheck.split(",");

    var isRolePresent = false;
    if (roleClaims != null) {
        for (i = 0; i < roleClaims.length; i++) {
            if (roleToCheckArray.indexOf(roleClaims[i]) != -1) {
                isRolePresent = true;
                break;
            }
        }
    }
    if (!isRolePresent) {
        gs.error("Azure Monitor Push ConnectorError: Role Validation has failed, so event will not be created. Roles Received  are " + roleClaims + " and RoleToCheck = " + roleToCheck);
    }


    return isRolePresent;
}

//// get CI sysid given AMP alert's target resource id
function getCmdbCiAndNameFromObjectId(targetResource) {
    var ci_gr = new GlideRecord("cmdb_ci_vm_object");
    ci_gr.addQuery("object_id", targetResource);
    ci_gr.query();
    if (ci_gr.next()) {
        return [ci_gr.sys_id, ci_gr.name];
    }

    return ["", ""];
}