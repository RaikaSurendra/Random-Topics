try {
  var request = new sn_ws.RESTMessageV2('x_hidsr_integratio.SEP API', 'Identity Authenticate');
  var bodyRequest = {};
  bodyRequest.username = gs.getProperty("x_hidsr_integratio.sepmApiUsername");
  bodyRequest.password = gs.getProperty("x_hidsr_integratio.sepmApiPassword");
  bodyRequest.domain = "";
  request.setRequestBody(JSON.stringify(bodyRequest));
  var response = request.execute();
  var responseBody = response.getBody();
  var httpStatus = response.getStatusCode();
  gs.print(JSON.parse(responseBody).token);
  gs.setProperty("x_hidsr_integratio.sepmApiToken",JSON.parse(responseBody).token)
} catch (ex) {
  var message = ex.message;
}

try {
  var request = new sn_ws.RESTMessageV2('x_hidsr_integratio.SEP API', 'Get Computer Information');
  request.setRequestHeader("Accept","Application/json");
  request.setRequestHeader("Authorization","Bearer be0354fa-eeeb-4d63-b157-2b6db8680a08");
  request.setStringParameterNoEscape("pageSize","100");
  request.setStringParameterNoEscape("verbose",true);
  request.setStringParameterNoEscape("pageIndex","1");
  var response = request.execute();
  var responseBody = response.getBody();
  var httpStatus = response.getStatusCode();
  gs.print(JSON.parse(responseBody));
} catch (ex) {
  var message = ex.message;
}

try {
  var request = new sn_ws.RESTMessageV2('x_hidsr_integratio.SEP API', 'Get AVDEF Latest');
  request.setRequestHeader("Accept","Application/json");
  request.setRequestHeader("Authorization","Bearer be0354fa-eeeb-4d63-b157-2b6db8680a08");
  var response = request.execute();
  var responseBody = response.getBody();
  var httpStatus = response.getStatusCode();
  gs.print(JSON.parse(responseBody).publishedBySEPM);
  var datSTr = JSON.parse(responseBody).publishedBySEPM;
  var datSTr = "6/11/2020 rev. 22";
  var firstArr = datSTr.split(' rev. ');
  var dateArr = firstArr[0].split('/');
  var revision = "0"+firstArr[1];
  var yy = dateArr[2].slice(-2);
  var mm = dateArr[0].length > 1 ? dateArr[0] : "0"+dateArr[0];
  var dd = dateArr[1].length > 1 ? dateArr[1] : "0"+dateArr[1];
  yy+mm+dd+revision;

} catch (ex) {
  var message = ex.message;
}
try {
  var datSTr = "6/11/2020 rev. 22";
  var firstArr = datSTr.split(' rev. ');
  var dateArr = firstArr[0].split('/');
  var revision = "0"+firstArr[1];
  var yy = dateArr[2].slice(-2);
	var mm = dateArr[0].length > 1 ? dateArr[0] : "0"+dateArr[0];
  var dd = dateArr[1].length > 1 ? dateArr[1] : "0"+dateArr[1];
  yy+mm+dd+revision;
} catch (ex) {
  var message = ex.message;
}
