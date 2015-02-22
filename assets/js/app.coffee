
app = angular.module 'irori', ['ngResource']

# Use '{$ $}' instead of '{{ }}' that is used by pongo2.
app.config ($interpolateProvider) ->
  $interpolateProvider.startSymbol '{$'
  $interpolateProvider.endSymbol '$}'

app.factory 'Project', [
  '$resource', ($resource) ->
    $resource '/api/projects.json', {}, {
      query: {method:'GET', isArray:true}
    }]

app.controller 'ProjectsCtrl', [
  'Project', (Project) ->
    this.projects = Project.query()
    this.project = new Project( Name: "")
    ctrl = this
    this.addProject = () ->
      console.log("addProject: ", ctrl.project)
      ctrl.project.$save () ->
        ctrl.projects = Project.query()
  ]
