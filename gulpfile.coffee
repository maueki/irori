gulp    = require "gulp"
coffee  = require "gulp-coffee"
plumber = require "gulp-plumber"
notify  = require "gulp-notify"
bower   = require "main-bower-files"
filter  = require "gulp-filter"
concat  = require "gulp-concat"

gulp.task 'compile-coffee', ->
  gulp.src('assets/js/*.coffee')
    .pipe(plumber({errorHandler: notify.onError('<%= error.message %>')}))
    .pipe(coffee())
    .pipe gulp.dest('assets/js')

gulp.task 'default', ->
  gulp.watch(["assets/js/*.coffee"], ["compile-coffee"])

gulp.task 'create-libjs', ->
  jsFilter = filter '**/*.js'
  gulp
    .src bower()
    .pipe jsFilter
    .pipe concat 'lib.js'
    .pipe gulp.dest 'assets/js'
