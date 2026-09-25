// Добавь эту глобальную переменную в самый верх файла webroot/app.js к остальным:
let boardHistory = []

// Полностью заменяем функцию handleTurn в webroot/app.js:
async function handleTurn(playerMoveIdx) {
  if (lockBoard) return
  if (playerMoveIdx !== -1 && boardState[playerMoveIdx] !== 0) return
  lockBoard = true
  let humanPassedThisTurn = false

  // Сохраняем текущую доску в историю ПЕРЕД ходом
  boardHistory.push([...boardState])
  if (boardHistory.length > 6) boardHistory.shift() // Храним последние 6 полуходов

  if (playerMoveIdx !== -1) {
    boardState[playerMoveIdx] = 1
    renderStones()
    appendToLog('Вы', playerMoveIdx)
    moveCount++
    moveCounterEl.innerText = moveCount
    statusTextEl.innerText = 'ИИ рассчитывает ход...'
  } else {
    humanPassedThisTurn = true
    appendToLog('Вы', -1)
    moveCount++
    moveCounterEl.innerText = moveCount
    statusTextEl.innerText = 'Вы пасанули. Ожидание ИИ...'
  }

  try {
    const response = await fetch('/move', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        board: boardState,
        humanScore: humanScore,
        aiScore: aiScore,
        lastMove: playerMoveIdx,
        moveCount: moveCount,
        maxMoves: MaxMoves,
        humanPassed: humanPassedThisTurn,
        boardHistory: boardHistory, // ОТПРАВЛЯЕМ ИСТОРИЮ НА СЕРВЕР
      }),
    })
    const data = await response.json()
    boardState = data.newBoard
    humanScore = data.humanScore
    aiScore = data.aiScore
    appendToLog('ИИ', data.move)
    moveCount++
    moveCounterEl.innerText = moveCount
    logTimeMs = data.timeMs || 0
    logTimeEl.innerText = logTimeMs.toFixed(2) + ' мс'
    logNodesEl.innerText = (data.nodes || 0).toLocaleString('ru-RU')
    humanScoreEl.innerText = humanScore
    aiScoreEl.innerText = aiScore
    renderStones()

    // Сохраняем доску после ответа ИИ в историю
    boardHistory.push([...boardState])
    if (boardHistory.length > 6) boardHistory.shift()

    if (data.gameOver) {
      let winMsg = ''
      if (humanScore > aiScore)
        winMsg = `<span style='color:#33ff33;'>Вы победили! (${humanScore}:${aiScore})</span>`
      else if (aiScore > humanScore)
        winMsg = `<span style='color:#ff3333;'>ИИ победил! (${aiScore}:${humanScore})</span>`
      else winMsg = "<span style='color:#ffaa00;'>Ничья!</span>"
      statusTextEl.innerHTML = `<b>${data.reason}</b><br>${winMsg}`
      lockBoard = true
      return
    }
    statusTextEl.innerText =
      data.move === -1 ? 'ИИ объявил ПАС! Ваш ход.' : 'Ваш ход.'
  } catch (err) {
    console.error(err)
    statusTextEl.innerText = 'Ошибка сервера.'
  }
  {
    if (
      !statusTextEl.innerHTML.includes('победил') &&
      !statusTextEl.innerHTML.includes('Ничья')
    )
      lockBoard = false
  }
}
