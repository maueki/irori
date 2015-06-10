gulp   = require "gulp"
coffee = require "gulp-coffee"

gulp.task 'compile-coffee', ->
  gulp.src('assets/js/*.coffee')
    .pipe(coffee())
    .pipe gulp.dest('assets/js')

gulp.task 'default', ->
  gulp.watch(["assets/js/*.coffee"], ["compile-coffee"])
