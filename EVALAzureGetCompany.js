EVAL(javascript: var rtrn;
  var gr = new GlideRecord('core_company'); gr.addQuery('name', 'Microsoft'); gr.query();
  if (gr.next()) {
    rtrn = gr.sys_id;
  })
