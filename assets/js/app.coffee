
app = angular.module 'irori', ['ngResource']

# In AngularJS, Use '{$ $}' instead of '{{ }}' that is used by pongo2.
app.config ($interpolateProvider) ->
  $interpolateProvider.startSymbol '{$'
  $interpolateProvider.endSymbol '$}'

app.factory 'Project', [
  '$resource', ($resource) ->
    $resource '/api/projects', {}, {
      query: {method:'GET', isArray:true}
    }]

app.controller 'ProjectsCtrl', [
  'Project',  '$scope', (Project, $scope) ->
    $scope.projects = Project.query()
    $scope.project = new Project( Name: "")
    ctrl = this
    this.addProject = () ->
      console.log("addProject: ", $scope.project)
      $scope.project.$save () ->
        $scope.projects = Project.query()
  ]

app.factory 'Page', [
  '$resource', ($resource) ->
    $resource '/api/pages/:Id', {Id:'@Id'}, {
      }]

app.controller 'PageCreateCtrl', [
  'Page', '$window', '$scope', (Page, $window, $scope) ->
    this.addPage = () ->
      page = new Page($scope.page)
      page.$save().then (res) ->
        $window.location.href = '/wiki/' + res.Id
  ]

app.controller 'PageUpdateCtrl', [
  'Page', '$window', '$scope', (Page, $window, $scope) ->
    this.updatePage = () ->
      page = new Page($scope.page)
      page.$save().then (res) ->
        $window.location.href = '/wiki/' + res.Id
    this.load = (id) ->
      $scope.page = Page.get {'Id': id}
  ]

app.directive 'pageEditor', () ->
  {
    restrict: 'E'
    templateUrl: '/assets/html/page-editor.html'
    link: (scope, element) ->
      # FIXME
      setLeavingMessage('You\'re about to throw away this text without posting it.')

      timer = null

      scope.sendText = () ->
        $.ajax
          type: 'POST'
          url: '/markdown'
          data:
            text: $('#body-editor')[0].value
          success: (data) ->
            $('#output', element).html(data) 

      scope.$watch 'page.Article.Body', (value) ->
        clearTimeout timer
        timer = setTimeout scope.sendText, 2000
  }

