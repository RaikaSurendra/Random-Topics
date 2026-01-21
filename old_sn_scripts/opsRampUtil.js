var compositeEventParser = Class.create();
compositeEventParser.prototype = {
  initialize: function() {
    this = {
      //string variables initialize
      data: '',
      eventState: '',
      newServiceOutput: '',
      sourceEventUniqueID: '',
      recordCreated: '',
      //object variables initialize
      finalObj: {},
      filteredObj: {},
      //integer variables initialize
      filteredObjLength: 0,
      finalObjLength: 0,
      //boolean variables initialize
      combinedEventFlag: false
    }
  },

  cleanServiceOutput: function( /*string*/ data, /*state of the event*/ eventState, /*event sys id*/ eventUniqueID) {
    var offset, offset2 = data.length;
    this.sourceEventUniqueID = eventUniqueID;
    this.eventState = eventState;
    if (eventState == 'CRITICAL')
      offset = 10;
    if (eventState == 'WARNING')
      offset = 9;
    if (data.slice(-3) == 'END') {
      offset2 = parseInt(-3);
      this.data = data.slice(parseInt(offset), offset2);
    }
    if (data.slice(-3) != 'END') {
      data = data.replace(/[^,]*$/g, '');
      data = data.slice(0, -1);
      this.data = data.slice(parseInt(offset));
    }
    return this;
  },
  separateEvents: function( /*Optional*/ separator1, /*Optional*/ separator2, /*Single Alert have CRITICAL and WARNING*/ separator3) {
    var finalObj = {};
    var firstArray = this.data.split(separator1);
    firstArray.forEach(function(itrItem) {
      var secondArray = itrItem.split('=');
      var getValue = itrItem.split(separator2).slice(1).join(separator2);
      if (secondArray[1]) {
        finalObj[secondArray[0].trim()] = getValue.trim();
      }
    });
    this.finalObj = finalObj;
    return this;
  },

  filterEvents: function( /*array*/ valuesToFilter) {
    //var localFinalObj = this.finalObj;
    var localFinalObj = JSON.parse(JSON.stringify(this.finalObj));

    function filter(obj) {
      for (var propName in obj) {
        if (obj[propName] && typeof obj[propName] === 'object')
          filter(obj[propName]);
        if (obj[propName] === null || obj[propName] === undefined || valuesToFilter.indexOf(obj[propName]) > -1) {
          delete obj[propName];
        }
      }
    }
    filter(localFinalObj);
    this.filteredObjLength = Object.keys(localFinalObj).length;
    this.finalObjLength = Object.keys(this.finalObj).length;
    this.filteredObj = localFinalObj;
    return this;
  },
  createIndividualEvent: function() {

  },
  createCombinedEvent: function() {
    this.combinedEventFlag = true;

    function createFinalSO(obj, str) {
      for (var propName in obj) {
        str += propName + "=" + obj[propName] + ',';
      }
      str = str.slice(0, -1);
      str += " END";
      return str;
    }

    if (this.filteredObjLength > 0) {
      this.newServiceOutput = this.eventState + ": ";
      this.newServiceOutput = createFinalSO(this.filteredObj, this.newServiceOutput);
    }
    return this;

  },
  execute: function( /*table name of event */ tableName) {
    var grParentEvent = new GlideRecord(tableName);
    grParentEvent.get(this.sourceEventUniqueID);
    if (this.combinedEventFlag === true && this.filteredObjLength > 0) {
      var grCombinedChildEvent = new GlideRecord(tableName);
      grCombinedChildEvent.newRecord();
      grCombinedChildEvent.eventreportedat = grParentEvent.eventreportedat;
      grCombinedChildEvent.hostalias = grParentEvent.hostalias;
      grCombinedChildEvent.hostgroupalias = grParentEvent.hostgroupalias;
      grCombinedChildEvent.hostname = grParentEvent.hostname;
      grCombinedChildEvent.longserviceoutput = grParentEvent.longserviceoutput;
      grCombinedChildEvent.servicedescription = grParentEvent.servicedescription;
      grCombinedChildEvent.servicedowntime = grParentEvent.servicedowntime;
      grCombinedChildEvent.serviceperfdata = grParentEvent.serviceperfdata;
      grCombinedChildEvent.servicestate = grParentEvent.servicestate;
      grCombinedChildEvent.event_operation_type = grParentEvent.event_operation_type;
      grCombinedChildEvent.serviceoutput = this.newServiceOutput;
      grCombinedChildEvent.interface_name = "Parsed";
      grCombinedChildEvent.insert();
      this.recordCreated = grCombinedChildEvent.getUniqueValue();
    }
    return this;
  },
  type: 'compositeEventParser'
};
