
app = angular.module('irori', [])
app.config ($interpolateProvider) ->
  $interpolateProvider.startSymbol '{$'
  $interpolateProvider.endSymbol '$}'

app.controller('ProjectsController', ['$http', ($http) ->
  ctrl = this
  ctrl.projects = []
  $http.get('/api/projects.json').success (data) ->
    ctrl.projects = data
  ])
