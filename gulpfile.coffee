gulp    = require "gulp"
coffee  = require "gulp-coffee"
plumber = require "gulp-plumber"
notify  = require "gulp-notify"

gulp.task 'compile-coffee', ->
  gulp.src('assets/js/*.coffee')
    .pipe(plumber({errorHandler: notify.onError('<%= error.message %>')}))
    .pipe(coffee())
    .pipe gulp.dest('assets/js')

gulp.task 'default', ->
  gulp.watch(["assets/js/*.coffee"], ["compile-coffee"])
