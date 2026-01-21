var sub = gs.getProperty('glide.knowman.submission.workflow');

if (sub == 'true')
  submitCandidate();
else
  submitDirect();
/**
 * SRA765-STRY0019590-ENHC0010661 (IR007) Draft KB Checkbox
 * [submitDirect Will check if there is any exisitng KB Article related to this incident which is in Draft, if not will create a new one]
 * @method submitDirect This method was earlier trying to create KB in stub Knowledge Base i.e dfc19531bf2021003f07e2c1ac0739ab
 * With new system property to store the sys_id of 'Known Errors'
 * @return {[GlideRecord]}     [glideRecord Object of the KB which was updated or created]
 */
function submitDirect() {
  var kb_base = gs.getProperty("glide.knowman.incident_known_errors_kb", "4ac985c837bd2640855b19a543990e88");
  var kb = new GlideRecord("kb_knowledge");
  kb.addQuery('source',current.getValue('sys_id'));
  kb.addQuery('kb_knowledge_base',kb_base);
  kb.addQuery('u_knowledge_type','Known Error');
  kb.addQuery('workflow_state','draft');
  kb.addActiveQuery();
  kb.orderByDesc('sys_created_on');
  kb.setLimit(1);
  kb.query();
  while(kb._next()){
    var localText;
    kb.setValue('short_description', current.getValue('short_description'));
    if(current.getValue('short_description') != kb.getValue('short_description')){
      localText += "</br> Short Description "+ current.getDisplayValue('short_description');
    }
    localText = kb.getValue('text') + current.comments.getHTMLValue();
    localText += "</br> Closure Information ";
    localText += "</br> Close Code "+ current.getDisplayValue('close_code');
    localText += "</br> Close Notes "+ current.getDisplayValue('close_notes');
    localText += "</br> Resolved At "+ current.getDisplayValue('resolved_at');
    kb.setValue('text',localtext);
    kb.update();
  }

  if(kb.getRow)
  kb.source = current.sys_id;
  kb.short_description = current.short_description;
  kb.sys_domain = current.sys_domain;
  kb.text = current.comments.getHTMLValue();
  kb.workflow_state = 'draft';
  //kb.kb_knowledge_base = gs.getProperty("glide.knowman.task_kb", "dfc19531bf2021003f07e2c1ac0739ab");
  //9feadb89db55b3486ddf5c88dc961908
  if (current.major_incident_state == "approved") kb.kb_knowledge_base = gs.getProperty("glide.knowman.major_incident_kb", "332bdb4ddb55b3486ddf5c88dc96199a");
  if (current.major_incident_state != "approved") kb.kb_knowledge_base = gs.getProperty("glide.knowman.incident_kb", "9feadb89db55b3486ddf5c88dc961908");

  kbSysId = kb.insert();
  if (kbSysId) {
    current.setValue('u_incident_knowledge_article', kbSysId);
    current.update();
    gs.addInfoMessage(gs.getMessage('Knowledge Article created: {0} based on closure of Incident: {1}', [kb.number, current.number]));
  }
}

function submitCandidate() {
  var gr = new GlideRecord('kb_submission');
  gr.parent = current.sys_id;
  gr.short_description = current.short_description;
  gr.sys_domain = current.sys_domain;
  gr.text = current.comments.getHTMLValue();
  gr.insert();
  gs.addInfoMessage(gs.getMessage('Knowledge Submission created: {0} based on closure of Incident: {1}', [gr.number, current.number]));
}
