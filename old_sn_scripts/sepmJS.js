var sepmIntegrationUtil = (function() {
  'use strict';
  //set REST and Import Set Related Variables
  var targetImportSetTable = 'x_hidsr_integratio_sepm_import_set';
  var restApi = 'x_hidsr_integratio.SEPM API';
  var identityMethod = 'Identity Authenticate';
  var getDefVersion = 'Get AVDEF Latest';
  var getComputers = 'Get Computer Information';
  //set user credentials for SEPM
  var spemIntPassword = gs.getProperty("x_hidsr_integratio.sepmApiPassword");
  var spemIntUsername = gs.getProperty("x_hidsr_integratio.sepmApiUsername");
  var spemIntToken = gs.getProperty("x_hidsr_integratio.sepmApiToken");

  //hoisting transcational values
  var token = '';
  /**
   * [_pad description]
   * @method      _pad
   * @param       {[type]} n     [description]
   * @param       {[type]} width [description]
   * @param       {[type]} z     [description]
   * @constructor
   * @return      {[type]} [description]
   */
  function _pad(n, width, z) {
    z = z || '0';
    n = n + '';
    return n.length >= width ? n : new Array(width - n.length + 1).join(z) + n;
  }
  /**
   * SEPM identity endpoint to provide the token, user credentials are set in the
   * properties
   * @return {[type]} [description]
   */
  function _getToken() {
    // Basic SN Rest call to SEPM, for token fetch
    try {
      var request = new sn_ws.RESTMessageV2(restApi, identityMethod);
      var bodyRequest = {};
      bodyRequest.username = spemIntUsername;
      bodyRequest.password = spemIntPassword;
      bodyRequest.domain = "";
      request.setRequestBody(JSON.stringify(bodyRequest));
      var response = request.execute();
      var responseBody = response.getBody();
      var httpStatus = response.getStatusCode();
      gs.setProperty("x_hidsr_integratio.sepmApiToken", JSON.parse(responseBody).token);
      token = (JSON.parse(responseBody).token);
      return token;
    } catch (e) {
      gs.error("Error Fetching Token from SEPM : " + e);
    } finally {
      gs.info("getToken from SEPM completed");
    }
  }
  /**
   * [getAVDef description]
   * @method getAVDef
   * @return {[type]} [description]
   */
  function getAVDef() {

    // Use token to get the AV definition Version
    try {
      token = _getToken();
      var tokenString = "Bearer " + token;
      var request = new sn_ws.RESTMessageV2(restApi, getDefVersion);
      request.setRequestHeader("Accept", "Application/json");
      request.setRequestHeader("Authorization", tokenString);
      var response = request.execute();
      var responseBody = response.getBody();
      var httpStatus = response.getStatusCode();
      gs.info(JSON.parse(responseBody).publishedBySEPM);
      var datSTr = JSON.parse(responseBody).publishedBySEPM;
      var firstArr = datSTr.split(' rev. ');
      var dateArr = firstArr[0].split('/');
      var revision = _pad(firstArr[1],3);
      var yy = dateArr[2].slice(-2);
      var mm = dateArr[0].length > 1 ? dateArr[0] : "0" + dateArr[0];
      var dd = dateArr[1].length > 1 ? dateArr[1] : "0" + dateArr[1];
      gs.addInfoMessage(yy + mm + dd + revision);
      gs.setProperty("x_hidsr_integratio.avDefCurrent",yy + mm + dd + revision);
    } catch (e) {
      gs.error("Version Fetch Completed with Error" + e);
    } finally {
      gs.info("Version Fetch Completed");
    }
  }
  /**
   * [getComputerInfo description]
   * @method getComputerInfo
   * @param  {[integer]}        pageSize  [description]
   * @param  {[type]}        verbose   [description]
   * @param  {[type]}        pageIndex [description]
   * @return {[type]}        [description]
   */
  function getComputerInfo(pageSize, verbose, pageIndex) {
    // Fetch Computers information from SEPM
    // gaurd the parameters passed
    pageSize = (typeof pageSize !== 'undefined') ? pageSize : 100;
    verbose = (typeof verbose !== 'undefined') ? verbose : true;
    pageIndex = (typeof pageIndex !== 'undefined') ? pageIndex : 1;

    try {
      token = _getToken();
      var tokenString = "Bearer " + token;
      var request = new sn_ws.RESTMessageV2(restApi, getComputers);
      request.setRequestHeader("Accept", "Application/json");
      request.setRequestHeader("Authorization", tokenString);
      request.setStringParameterNoEscape("pageSize", pageSize);
      request.setStringParameterNoEscape("verbose", verbose);
      request.setStringParameterNoEscape("pageIndex", pageIndex);
      var response = request.execute();
      var responseBody = response.getBody();
      var httpStatus = response.getStatusCode();
      return responseBody;
    } catch (e) {}
  }
  /**
   * [runImport description]
   * @method runImport
   * @return {[type]}  [description]
   */
  function runImport() {
    var totalPages;
    var totalElements;
    var lastPage;
    var currentPage;
    var startPage = 1;

    do {
      var runIter = JSON.parse(getComputerInfo(100, true, startPage));
      if (runIter.number == 0) {
        totalPages = runIter.totalPages;
        totalElements = runIter.totalElements;
        lastPage = runIter.lastPage;
        currentPage = runIter.number + 1;
      }
      runIter.content.forEach(function(item) {
        var importHost = new GlideRecord(targetImportSetTable);
        importHost.newRecord();
        importHost.av_def_set_version = item.avDefsetVersion ? item.avDefsetVersion : "";
        importHost.online_status = item.onlineStatus == 1 ? true : false;
        importHost.hostname = item.computerName ? (item.computerName).toLowerCase() : "";
        importHost.ip_address = item.ipAddresses[0] ? item.ipAddresses[0] : "";
        importHost.is_infected = item.infected == 1 ? true : false;
        importHost.os_function = item.osFunction;
        importHost.os_name = item.osName;
        importHost.sepm_unique_id = item.uniqueId;
        importHost.serial_number = item.serialNumber ? item.serialNumber : "";
        importHost.json_payload = JSON.stringify(item, " ", 2);
        importHost.insert();
      });
      startPage++;
    }
    while (startPage <= totalPages);
  }
  /**
   *
   */
  return {
    'getAVDef': getAVDef,
    'getComputerInfo': getComputerInfo,
    'runImport': runImport
  };
})();
