marked.setOptions
  gfm: true
  tables: true
  breaks: false
  pedantic: false
  sanitize: true
  smartLists: true
  smartypants: false
  langPrefix: ''
  highlight: (code) ->
    return highlightAuto(code).value

edit = new Vue {
  el: '#edit'
  data: {
    page: {
      article: {
        title: ''
        body: ''
      }
      access: ''
      author: ''
      groups: []
      projects: []
    }
    groups: []
    projects: []
    pageOutput: ''
  }
  filters:
    marked: marked
  methods: {
    getPage: (pageId) ->
      $.ajax
        type: 'GET'
        url: '/api/pages/' + pageId
        success: (data) ->
          data
    getProjects: ->
      $.ajax
        type: 'GET'
        url: '/api/projects'
        success: (data) ->
          data
    getGroups: ->
      $.ajax
        type: 'GET'
        url: '/api/groups'
        success: (data) ->
          data
    postPage: ->
      page = JSON.parse(JSON.stringify(@page)) #FIXME
      page.projects = (p.id for p in @projects when p.enabled)
      page.groups = (g.id for g in @groups when g.enabled)
      $.ajax
        type: 'POST'
        url: '/api/pages/' + $('#edit').data('config').pageId
        data: JSON.stringify(page)
        success: (res) ->
          window.location.href = '/docs/' + res.id
  }
  created: ->
    # FIXME
    setLeavingMessage('You\'re about to throw away this text without posting it.')

  compiled: ->
    @timer = null
    hljs.initHighlightingOnLoad()

    procs = []
    procs.push(
      @getGroups().then (data) =>
        @$data.$set('groups',data)
    )

    pageId = $('#edit').data('config').pageId
    if pageId != ''
      procs.push(
        @getPage(pageId).then (data) =>
          @$data.$set('page', data)
      )

    procs.push(
      @getProjects().then (data) =>
        @$data.$set('projects', data)
    )

    $.when(procs...)
      .done =>
        for g in @groups
          if g.id in @page.groups
            g.$set('enabled', true)
        for p in @projects
          if p.id in @page.projects
            p.$set('enabled', true)

    @$watch('page.article.body', (newval, oldval) =>
      clearTimeout @timer
      @timer = setTimeout @sendText, 2000
    )
}
