
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
    $resource '/api/pages/:pageId', {pageId:'@id'}, {
      }]

app.controller 'PageCreateCtrl', [
  'Page', 'Project', 'Group', '$window', '$scope', (Page, Project, Group, $window, $scope) ->
    $scope.groups = Group.query()
    $scope.projects = Project.query()

    this.addPage = () ->
      page = new Page($scope.page)
      if page.Access == "group"
        page.Groups = (g.id for g in $scope.groups when g.enabled)
      page.$save().then (res) ->
        $window.location.href = '/wiki/' + res.Id
  ]

app.controller 'PageUpdateCtrl', [
  'Page', 'Group', '$window', '$scope', (Page, Group, $window, $scope) ->
    $scope.groups = Group.query()

    this.updatePage = () ->
      page = new Page($scope.page)
      page.$save().then (res) ->
        $window.location.href = '/wiki/' + res.Id

    this.load = (id) ->
      $scope.page = Page.get {'pageId': id}
      $scope.page.$promise.then (page)->
        for g in $scope.groups
          if g.id in page.Groups
            g.enabled = true
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

app.directive 'pageSidebar', () ->
  {
    restrict: 'E'
    templateUrl: '/assets/html/page-sidebar.html'
    controller: ['Group', '$scope', (Group, $scope) ->
      ]
    link: ($scope, element) ->
  }

app.factory 'User', [
  '$resource', ($resource) ->
    $resource '/api/users/:userId', {Id: '@userId'}, {
    }]

app.factory 'Group', [
  '$resource', ($resource) ->
    $resource '/api/groups/:groupId', {groupId: '@id'}, {
      update: {method: 'PUT'}
    }]

app.controller 'GroupCtrl', [
  'Group', '$scope', (Group, $scope) ->
    $scope.groups = Group.query()
    $scope.group = new Group( Name: "")

    this.addGroup = () ->
      console.log('add group: ', $scope.group)
      $scope.group.$save ()->
        $scope.group.Name = ''
        $scope.groups = Group.query()
  ]

app.controller 'EditGroupCtrl', [
  'Group', 'User', '$window', '$scope', (Group, User, $window, $scope) ->
    this.load = (id) ->
      $scope.group = Group.get({'groupId': id})

      User.query().$promise.then (users) ->
        $scope.users = ( {
          Id : user.Id
          Name : user.Name
          enabled : user.Id in $scope.group.Users
        } for user in users )

    this.submit = () ->
      console.log $scope.group.Name
      $scope.group.Users = ( user.Id for user in $scope.users when user.enabled)
      $scope.group.$update().then () ->
        $window.location.href = '/admin/groups'
  ]
