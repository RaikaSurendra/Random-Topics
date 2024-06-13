var interactionUtil = (function() {
	/**
	 * hoisting default variables like Table names
	 */
	var interactionMetaBlobTable = "interaction_json_blob";
	var interactionTable = "interaction";
	var contextTable = "v_interaction_context";
	var conversationTable = "sys_cs_conversation";
	var conversationMessageTable = "sys_cs_message";
	var conversationTasks = "sys_cs_conversation_task";
	var localMap = {};

	/**
	 * [_getInteractionBlob description]
	 * @method      _getInteractionBlob
	 * @constructor
	 * @return      {[type]}            [description]
	 */
	function _getInteractionBlob(uniqueId) {
		return _getGlideRecordObj(interactionMetaBlobTable, uniqueId);
	}
	/**
	 * [_getContext description]
	 * @method      _getContext
	 * @constructor
	 * @return      {[type]}    [description]
	 */
	function _getContext(uniqueId) {

	}
	/**
	 * [_getConversation description]
	 * @method      _getConversation
	 * @constructor
	 * @return      {[type]}         [description]
	 */
	function _getConversation(uniqueId) {
		return _getGlideRecordObj(conversationTable, uniqueId);
	}
	/**
	 * [_getConversationMessages description]
	 * @method      _getConversationMessages
	 * @constructor
	 * @return      {[type]}                 [description]
	 */
	function _getConversationMessages(conversationUniqueID) {
		var messageList = _getGlideRecordSysIDList(conversationMessageTable, "conversation=" + conversationUniqueID);
		return messageList;
	}
	/**
	 * [_getGlideRecordObj description]
	 * @method      _getGlideRecordObj
	 * @constructor
	 * @return      {[type]}           [description]
	 */
	function _getGlideRecordObj(tableName, uniqueId) {
		if (!tableName) return "No Table Name Passed";
		var glideRecordObj = new GlideRecord(tableName);
		if (!glideRecordObj.isValid()) return "Invalid Table";
		if (!uniqueId) return "Unique ID not passed";
		glideRecordObj.get(uniqueId);
		return glideRecordObj;
	}
	/**
	 * [_getGlideRecordSysIDList description]
	 * @method      _getGlideRecordSysIDList
	 * @constructor
	 * @return      {[type]}                 [description]
	 */
	function _getGlideRecordSysIDList(tableName, encodedQuery) {
		var localArray = [];
		if (!tableName) return "No Table Name Passed";
		var glideRecordObj = new GlideRecord(tableName);
		if (!glideRecordObj.isValid()) return "Invalid Table";
		if (!encodedQuery) return "Encoded query not passed";
		glideRecordObj.addEncodedQuery(encodedQuery);
		glideRecordObj.query();
		glideRecordObj.getRowCount();
		gs.addInfoMessage(glideRecordObj.getEncodedQuery());
		while (glideRecordObj._next()) {
			localArray.push(glideRecordObj.getUniqueValue());
		}
		return localArray;
	}
	/**
	 * [_getInteraction description]
	 * @method      _getInteraction
	 * @constructor
	 * @return      {[type]}        [description]
	 */
	function _getInteraction(uniqueId) {
		return _getGlideRecordObj(interactionTable, uniqueId);
	}
	/**
	 * [deleteMessages description]
	 * @method deleteMessages
	 * @return {[type]}       [description]
	 */
	function deleteMessages(map) {
		var csMessage = new GlideRecord(conversationMessageTable);
		if (map["sys_cs_conversation"]) {
			csMessage.addQuery('conversation', map["sys_cs_conversation"]);
			csMessage.query();
      gs.addInfoMessage(csMessage.getEncodedQuery());
			csMessage.deleteMultiple();
		}
	}
	/**
	 * [getMap description]
	 * @method getMap
	 * @return {[type]} [description]
	 */
	function getMap(interactionSysId) {
		var interactionObj = _getInteraction(interactionSysId);
		var contextObj = interactionObj.context;
		var conversationUniqueID = interactionObj.getValue("channel_metadata_document");
		var conversationObj = _getConversation(conversationUniqueID);
		var interactionBlobUniqueID = interactionObj.getValue("context_document");
		var interactionBlobObj = _getInteractionBlob(interactionBlobUniqueID);
		var messageList = _getConversationMessages(conversationObj.getUniqueValue());
		return {
			"sys_cs_conversation": conversationObj,
			"messageList": messageList
		};
	}

	return {
		deleteMessages: deleteMessages,
		getMap: getMap
	};

})();
