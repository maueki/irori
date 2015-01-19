$ ->
  timer = null

  send_text = () ->
    $.ajax
      type: 'POST'
      url: '/markdown'
      data:
        text: $('#editor textarea')[0].value
      success: (data) ->
        $('#editor #output').html(data)

  ta = $('#editor textarea')[0]
  ta.addEventListener 'input', () ->
    clearTimeout(timer)
    timer = setTimeout(send_text, 2000)
