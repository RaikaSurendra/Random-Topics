var DeleteChatHistory = Class.create();
DeleteChatHistory.prototype = {
  initialize: function() {},
  deleteChats: function(chatTaskObj) {
    var sysID = chatTaskObj.sys_id.toString();
    var gr = new GlideRecord('sys_history_line');
    //gr.get('a75a25011b2060d0608786e07e4bcb2e');
    gr.addEncodedQuery('set.id=' + sysID + '^field=work_notes^ORfield=comments');
    // we should add a created date filter here for table rotation, are chats always closed within 7 days?
    //^sys_created_onONLast 7 days@javascript:gs.beginningOfLast7Days()@javascript:gs.endOfLast7Days()
    gr.query();
    while (gr.next()) {
      var fieldVal = gr["new"];
      //gs.log('fieldVal: '+fieldVal, 'JWI115');
      //gs.log('documentKey: '+gr.set.id, 'JWI115');
      var fieldName = gr.field;
      //gs.log('fieldName : '+fieldName, 'JWI115');
      //Query for and delete the 'sys_audit' record
      var aud = new GlideRecord('sys_audit');
      aud.addQuery('documentkey', gr.set.id);
      aud.addQuery('fieldname', fieldName);
      aud.addQuery('newvalue', fieldVal);
      aud.query();
      if (aud.next()) {
        //gs.log("sys_audit: "+aud.sys_id.toString(), 'JWI115');
        aud.deleteRecord();
      }
      //Query for and delete the 'sys_journal_field' record (if applicable)
      var je = new GlideRecord('sys_journal_field');
      je.addQuery('element_id', gr.set.id);
      je.addQuery('element', fieldName);
      je.addQuery('value', fieldVal);
      je.query();
      if (je.next()) {
        //gs.log("sys_journal_field: "+je.sys_id.toString(), 'JWI115');
        je.deleteRecord();
      }
      //Delete the 'sys_history_line' record
      gr.deleteRecord();
    }
  },

  type: 'DeleteChatHistory'
};
