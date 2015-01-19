$ ->
  timer = null

  send_text = () ->
    console.log($('#editor textarea')[0].value)

  ta = $('#editor textarea')[0]
  ta.addEventListener 'input', () ->
    clearTimeout(timer)
    timer = setTimeout(send_text, 2000)
