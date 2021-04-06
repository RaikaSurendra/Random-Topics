promoteKBtoReview();
/**
 * SRA765-STRY0019590-ENHC0010661 (IR007) Draft KB Checkbox
 * function to promote the Knowledge Article to Review and attach the Approvers from the Resolver Group
 * @method promoteKBtoReview
 */
function promoteKBtoReview() {
  var kb_base = gs.getProperty("glide.knowman.incident_known_errors_kb", "4ac985c837bd2640855b19a543990e88");
  var kb_category = gs.getProperty("glide.knowman.incident_known_errors_kb_category", "2fd5f8181b0be49470d0a792f54bcbb9");
  var kbArticle = new GlideRecord("kb_knowledge");
  kbArticle.addQuery('source', current.getValue('sys_id'));
  kbArticle.addQuery('kb_knowledge_base', kb_base);
  kbArticle.addQuery('kb_category', kb_category.toString());
  kbArticle.addQuery('u_knowledge_type', 'Known Error');
  kbArticle.addQuery('workflow_state', 'draft');
  kbArticle.addActiveQuery();
  kbArticle.orderByDesc('sys_created_on');
  kbArticle.setLimit(1);
  kbArticle.query();
  while (kbArticle._next()) {
    kbArticle.setValue('workflow_state', 'review');
    var wf = new Workflow();
    var vars = {};
    vars.u_sendapprovaltoresolvergroup = true;
    var wfId = wf.getWorkflowFromName("Knowledge - Approval Publish");
    wf.startFlow(wfId, kbArticle, "Knowledge - Approval Publish", vars);
    kbArticle.update();
  }
}
