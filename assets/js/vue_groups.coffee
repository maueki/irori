request = window.superagent

groups = new Vue {
  el: '#groups'
  data: {
    groups: []
    inputGourp: ''
  }
  methods: {
    addGroup: (e) ->
      e.preventDefault()
      request
        .post('/api/groups')
        .send({name: @inputGroup})
        .end (err, res) =>
          @inputGroup = ''
          @listGroup()
    listGroup: () ->
      request
        .get('/api/groups')
        .end (err, res) =>
          @groups = res.body
  }
  created: () ->
    @listGroup()
}
