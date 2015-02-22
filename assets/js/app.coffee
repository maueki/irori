
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
  ]
