request = window.superagent

pages = new Vue {
  el: '#pages'
  data: {
    pages: []
    ownpages: []
  }
  methods: {
    pageInfo: (page) ->
      request
        .get('/api/users/' + page.article.userId)
        .end (err, res) =>
          page.article.$set('user',  res.body)
  }
  compiled: ->
    request
      .get('/api/pages')
      .end (err, res) =>
        @pages = res.body
        for page in @pages
          @pageInfo(page)

    request
      .get('/api/pages/own')
      .end (err, res) =>
        @ownpages = res.body
        for page in @ownpages
          @pageInfo(page)
}
