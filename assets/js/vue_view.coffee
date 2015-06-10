request = window.superagent

marked.setOptions {
  gfm: true
  tables: true
  breaks: false
  pedantic: false
  sanitize: true
  smartLists: true
  smartypants: false
  langPrefix: ''
}

view = new Vue {
  el: '#view'
  data: {
    page: {
      id: ''
      article: {
        title: ''
        body: ''
      }
    }
    pagebody: ''
  }
  methods: {
  }
  created: () ->
    pageId = $('#view').data('config').pageId
    request
      .get('/api/pages/' + pageId)
      .end (err, res) =>
        @page = res.body
        @pagebody = marked(@page.article.body)
        $('pre code', $('#pagebody')).each (i, e) ->
            hljs.highlightBlock(e)
}
