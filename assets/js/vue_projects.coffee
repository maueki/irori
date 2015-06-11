request = window.superagent

projects = new Vue {
  el: '#projects'
  data: {
    project: {
      name: ''
    }
    projects: []
  }
  methods: {
    addProject: ->
      console.log @project.name
      request
        .post('/api/projects')
        .send({name: @project.name})
        .end (err, res) =>
          @update()

    update: ->
      request
        .get('/api/projects')
        .end (err, res) =>
          @projects = res.body
  }
  created: ->
    @update()
}
