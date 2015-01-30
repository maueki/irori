
$ ->
  $(document).on 'click', 'a[data-method]', ->
    link = $(@)
    method = link.data('method').toLowerCase()
    if not method? or method != 'post'
      return true

    # data-method='POST'
    href = link.attr('href')
    form = $('<form method="post" action="' + href + '" type="hidden" />')
    metadataInput = '<input name="_method" value="' + method + '" type="hidden" />'
    form.hide().append(metadataInput).appendTo('body')
    form.submit()
    return false
