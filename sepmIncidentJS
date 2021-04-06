/**
 * [Base SEPM Utility to create records in the SEPM Incident Import set]
 * @type {namespace pattern}
 */
var sepmIncidentUtil = {
    /**
     * [set all the default values and return the same]
     * @method
     * @return {[Object with all defaults as properties]} [description]
     */
    _defaultsSetter: function() {
      var localObj = {};
      localObj.sepmIncidentImportSet = "x_hidsr_integratio_sepm_incidents";
      localObj.ciTable = "cmdb_ci_win_server";
      localObj.encodedInfectedHostsQuery = "sys_class_name=cmdb_ci_win_server^u_infected=true^discovery_source=SEPM";
      localObj.encodedOutdatedAVHostsQuery = "sys_class_name=cmdb_ci_win_server^discovery_source=SEPM^u_av_version!=javascript:global.getSystemProperties('5a9eaca01ba5d410e54fa9b62a4bcb4e');";
      //return the object ⛳
      return localObj;
    },
		/**
		 * [description]
		 * @method
		 * @param  {[type]} option [description]
		 * @return {[type]} [description]
		 */
    _getConfigurationItems: function(option) {
      var returnHosts = [];
      var options = option ? option : "outdatedAVHosts";
      var defaultObj = x_hidsr_integratio.sepmIncidentUtil._defaultsSetter();
      var encodedString = (options == "outdatedAVHosts") ? defaultObj.encodedOutdatedAVHostsQuery.toString() : defaultObj.encodedInfectedHostsQuery.toString();
      var hostsGR = new GlideRecord(defaultObj.ciTable.toString());
      hostsGR.addEncodedQuery(encodedString);
      hostsGR.query();
      while (hostsGR._next()) {
        returnHosts.push(hostsGR.getUniqueValue());
      }
      return returnHosts;
    },
		/**
		 * [description]
		 * @method
		 * @return {[type]} [description]
		 */
    _createImportSetRecord: function(item, typeOfIncident) {
			var defaultObj = x_hidsr_integratio.sepmIncidentUtil._defaultsSetter();
			var newRecordImportSet = new GlideRecord(defaultObj.sepmIncidentImportSet.toString());
			newRecordImportSet.newRecord();
			newRecordImportSet.setValue('configuration_item', item.toString());
			if(typeOfIncident == "infectedHosts") newRecordImportSet.setValue("is_infected", true);
			if(typeOfIncident == "outdatedAVHosts") newRecordImportSet.setValue("is_avdef_outdated", true);
			newRecordImportSet.insert();
    },
    /**
     * [get list of onfected SEPM hosts and insert in SEPM Incident Import Set]
     * @method infectedHosts
     */
    infectedHosts: function() {
      var defaultObj = x_hidsr_integratio.sepmIncidentUtil._defaultsSetter();
      var infectedHostsArray = x_hidsr_integratio.sepmIncidentUtil._getConfigurationItems("infectedHosts");
      if (infectedHostsArray.length > 0) {
        infectedHostsArray.forEach(function(item) {
          x_hidsr_integratio.sepmIncidentUtil._createImportSetRecord(item, "infectedHosts");
        });
      }
			return infectedHostsArray;
    },
    /**
     * get list of outdated AV SEPM hosts and insert in SEPM Incident Import Set
     * @method
     * @return {[type]} [description]
     */
    outdatedAVHosts: function() {
      var defaultObj = x_hidsr_integratio.sepmIncidentUtil._defaultsSetter();
      var outdatedAVHostsArray = (x_hidsr_integratio.sepmIncidentUtil._getConfigurationItems("outdatedAVHosts"));
      if (outdatedAVHostsArray.length > 0) {
        outdatedAVHostsArray.forEach(function(item) {
					x_hidsr_integratio.sepmIncidentUtil._createImportSetRecord(item, "outdatedAVHosts");
        });
      }
			return outdatedAVHostsArray;
    }
	};
		
