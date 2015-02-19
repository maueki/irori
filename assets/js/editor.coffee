$ ->
  # FIXME
  setLeavingMessage('You\'re about to throw away this text without posting it.')

  timer = null

  send_text = () ->
    $.ajax
      type: 'POST'
      url: '/markdown'
      data:
        text: $('#body-editor')[0].value
      success: (data) ->
        $('#output').html(data)

  ta = $('#body-editor')[0]
  ta.addEventListener 'input', () ->
    clearTimeout(timer)
    timer = setTimeout(send_text, 2000)
