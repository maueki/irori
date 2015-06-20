request = window.superagent

marked.setOptions
  gfm: true
  tables: true
  breaks: false
  pedantic: false
  sanitize: true
  smartLists: true
  smartypants: false
  highlight: (code, lang) ->
    return hljs.highlightAuto(code, [lang]).value

view = new Vue
  el: '#view'
  data:
    page:
      id: ''
      article:
        title: ''
        body: ''
  filters:
    marked: marked
  created: () ->
    pageId = $('#view').data('config').pageId
    request
      .get('/api/pages/' + pageId)
      .end (err, res) =>
        @page = res.body
