
$ ->
  $('div[data-alert-type]').each ->
    type = this.getAttribute('data-alert-type')
    value = this.getAttribute('data-alert-value')

    if value?.length != 0
      switch type
        when 'danger', 'warning', 'info', 'success'
          displayAlert(this, type, value)

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


displayAlert = (parentElement, type, text) ->
  element = document.createElement('div')
  element.setAttribute('class', "alert alert-#{type}")
  element.setAttribute('role', 'alert')
  element.innerHTML = text

  if parentElement
    parentElement.appendChild(element)


@setLeavingMessage = (message) ->
  isChanged = false
  $(window).bind 'beforeunload', ->
    alert('beforeunload')
    if isChanged
      return message
    return

  $('form input, form select, form textarea').each ->
    this.addEventListener 'input', () ->
      if !isChanged
        isChanged = true
      return

  $('input[type=submit], button[type=submit]').click ->
    isChanged = false
    return
