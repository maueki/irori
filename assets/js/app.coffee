
app = angular.module 'irori', ['ngResource', 'ngMessages', 'ui.utils']

# In AngularJS, Use '{$ $}' instead of '{{ }}' that is used by pongo2.
app.config ($interpolateProvider) ->
  $interpolateProvider.startSymbol '{$'
  $interpolateProvider.endSymbol '$}'

app.factory 'Project', [
  '$resource', ($resource) ->
    $resource '/api/projects', {}, {
    }]

app.controller 'ProjectsCtrl', [
  'Project',  '$scope', (Project, $scope) ->
    $scope.projects = Project.query()
    $scope.project = new Project( name: "")
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
      if page.access == "group"
        page.groups = (g.id for g in $scope.groups when g.enabled)
      page.$save().then (res) ->
        $window.location.href = '/docs/' + res.id

      page.projects = (p.id for p in $scope.projects when p.enabled)
  ]

app.controller 'PageUpdateCtrl', [
  'Page', 'Project', 'Group', '$window', '$scope', (Page, Project, Group, $window, $scope) ->
    $scope.groups = Group.query()
    $scope.projects = Project.query()

    this.updatePage = () ->
      page = new Page($scope.page)
      if page.access == "group"
        page.groups = (g.id for g in $scope.groups when g.enabled)
      else
        page.groups = []

      page.projects = (p.id for p in $scope.projects when p.enabled)

      page.$save({pageId: page.id}).then (res) ->
        $window.location.href = '/docs/' + res.id

    this.load = (id) ->
      $scope.page = Page.get {'pageId': id}
      $scope.page.$promise.then (page)->
        for g in $scope.groups
          if g.id in page.groups
            g.enabled = true
        for p in $scope.projects
          if p.id in page.projects
            p.enabled = true
  ]

app.controller 'PageCtrl', [
  'Page', 'User', '$window', '$scope', (Page, User, $window, $scope) ->
    $scope.pages = Page.query (pages) ->
      for page in pages
        page.article.user = User.get {userId: page.article.userId}
    $scope.ownpages = Page.query({pageId:'own'})
  ]

app.controller 'PageSearchCtrl', [
  'Page', 'User', '$window', '$scope', (Page, User, $window, $scope) ->

    $scope.init = (query) ->
      $scope.pages = Page.query {q: query },  (pages) ->
        for page in pages
          page.article.user = User.get {userId: page.article.userId}
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

      scope.$watch 'page.article.body', (value) ->
        clearTimeout timer
        timer = setTimeout scope.sendText, 2000
  }

app.directive 'pageSidebar', () ->
  {
    restrict: 'E'
    templateUrl: '/assets/html/page-sidebar.html'
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
    $scope.group = new Group( name: "")

    this.addGroup = () ->
      console.log('add group: ', $scope.group)
      $scope.group.$save ()->
        $scope.group.name = ''
        $scope.groups = Group.query()
  ]

app.controller 'EditGroupCtrl', [
  'Group', 'User', '$window', '$scope', (Group, User, $window, $scope) ->
    this.load = (id) ->
      $scope.group = Group.get({'groupId': id})

      User.query().$promise.then (users) ->
        $scope.users = ( {
          id : user.id
          name : user.name
          enabled : user.id in $scope.group.users
        } for user in users )

    this.submit = () ->
      console.log $scope.group.name
      $scope.group.users = ( user.id for user in $scope.users when user.enabled)
      $scope.group.$update().then () ->
        $window.location.href = '/admin/groups'
  ]

app.controller 'UsersCtrl', [
  'User', '$scope', '$window', (User, $scope, $window) ->
    $scope.users = User.query()
    $scope.deleteUser = (id) ->
      User.delete {userId: id}, (res) ->
        $scope.users = User.query()
  ]

app.controller 'UserAddCtrl', [
  'User', '$scope', '$window', (User, $scope, $window) ->
    this.addUser = () ->
      console.log $scope.user
      user = new User($scope.user)
      user.$save().then (res) ->
        $window.location.href = '/admin/users'
  ]

app.factory 'Password', [
  '$resource', ($resource) ->
    $resource '/api/password', {}, {
      update: {method: 'PUT'}
      }]

app.controller 'UserPasswordController', [
  'Password', '$scope', '$window', (Password, $scope, $window) ->
    this.updatePassword = () ->
      password = new Password($scope.password)
      console.log password
      password.$update().then () ->
          $window.location.href = '/profile'
        ,() ->
          alert('パスワードの更新に失敗しました')
  ]

app.controller 'NavbarCtrl', [
  '$scope', '$window', ($scope, $window)->
    this.submit = () ->
      if $scope.inputquery.length != 0
        $window.location.href = '/docs?q=' + $scope.inputquery
  ]

