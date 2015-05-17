request = window.superagent

view = new Vue {
  el: '#view'
  data: {
    page: {
      id: ''
      article: {
        title: ''
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
    request
      .get('/api/pages/' + pageId + '/body')
      .end (err, res) =>
        @pagebody = res.text
        $('pre code', $('#pagebody')).each (i, e) ->
            hljs.highlightBlock(e)
}
