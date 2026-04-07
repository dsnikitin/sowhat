package message

import (
	"fmt"
	"strings"

	"github.com/dsnikitin/sowhat/internal/models"
)

const Introduction = `
👋 Привет, %s!

🤖 Я SoWhatBot - умный помощник для конспектирования встреч.

🕹️ Введи команду:
• /list - покажу твои встречи
• /get <id> - покажу транскрипцию встречи
• /find <слово/слова> - найду встречи по ключевым словам
• /chat <вопрос> - отвечу на вопрос по встречам
• /help - покажу справку, если что-то забыл

🎙️ Отправь голосовое сообщение или аудио файл - сделаю транскрипцию и резюме
💡 Напиши вопрос без команды для продолжения последнего диалога
`

const WelcomeBack = `
👋 Рад снова видеть тебя, %s!

🤖 Чем могу помочь?

🕹️ Введи команду:
• /list - покажу твои встречи
• /get <id встречи> - покажу транскрипцию встречи
• /find <слово/слова> - найду встречи по ключевым словам
• /chat <вопрос> - отвечу на вопрос по встречам
• /help - покажу справку, если что-то забыл

🎙️ Отправь голосовое сообщение или аудио файл - сделаю транскрипцию встречи и резюме
💡 Напиши вопрос без команды для продолжения последнего диалога
`

const Help = `
🕹️ команды:
• /start - зарегистрироваться 
• /list - показать список встреч
  опционально номер страницы: /list 5
• /get <id встречи> - показать транскрипцию встречи
• /find <слово/слова> - найти встречи по ключевым словам
  опционально номер страницы: /find доброе утро 5
• /chat <вопрос> - получить ответ на вопрос по встречам
• /help - показать это сообщение

🎙️ аудио/голосовое - сделать транскрипцию встречи и резюме
💡 сообщение без команды - продолжение последнего диалога
`

const MeetingRegistered = `
🚀 Встреча зарегистрирована и уже отправлена на обработку.
ID встречи %d.
Пришлю уведомление, когда она будет полностью обработана.
`

const MeettingTranscriptionCompleted = `
🔔 Встреча %d полностью обработана.
Используй команду /get <id встречи> чтобы посмотреть транскрипцию.
`

const IdentificationFailed = `
🚫 Разработчик запретил мне общаться с незнакомцами.
Используй команду /start чтобы подружиться.
`

const NoMeetings = `
📭 Встреч пока нет.
Отправь аудио файл или голосовое сообщение, чтобы начать.
`

const NoMoreMeetings = `
📭 Встреч больше нет.
`

const IncorrectMeetingID = `
⚠️ Укажи корректный ID любой встречи.
Например: /get 5
`

const IncorrectListPage = `
⚠️ Укажи корректный номер страницы списка.
Если не укажешь, то получишь первую.
Например: /list или /list 5
`

const TooMuchArguments = `
⚠️ Укажи корректное количество аргументов для команды.
Используй команду /help для справки.
`

const UnsupportedAudioFormat = `
⚠️ Отправь аудио-файл поддерживаемого формата.
%s.
`

const UnsupportedFileSize = `
⚠️ Отправь аудио-файл или голосовое размером от %d до %d байт.
`

const MeetingNotFound = `
🔍 Нет такой встречи.
Используй команду /list чтобы увидеть список встреч.
`

const EmptyFindQuery = `
💬 Напиши ключевое слово/слова для поиска.
Например: /find добрый день
`

const EmptyChatQuery = `
💬 Задай какой-нибудь вопрос по встречам.
Например: /chat обсуждали ли мы отпуск
`

const NoFilesForQuestion = `
📭 Нет обработанных встреч, чтобы ответить на вопрос.
Отправь аудио файл или голосовое сообщение или дождись обработки уже отправленных.
`

const MeetingsNotFound = `
📭 Ничего не нашел.
Попробуй поискать по другим ключевым словам.
`

const OperationFailed = `
⚠️ Произошла непредвиденная ошибка.
Попробуй повторить операцию.
`

const TooBusy = `
🐌 Похоже, у меня слишком много дел, и я не успеваю.
Попробуй повторить операцию позднее.
`

func MeeetingWithTranscript(meeting models.Meeting, dateFormat string, transcriptLength int) string {
	var transcript string
	if meeting.IsTranscriptionFailed {
		transcript = "⚠️ Возникли технические неполадки. Транскрипция не будет создана."
	} else if meeting.Transcript == nil {
		transcript = "⏳ Ещё создаю. Осталось совсем немного."
	} else {
		transcript = truncate(*meeting.Transcript, transcriptLength)
	}

	return fmt.Sprintf(
		"🗓️ Встреча: %d\n📅 Дата: %s\n\n📜 Транскрипция:\n%s",
		meeting.ID,
		meeting.CreatedAt.Format(dateFormat),
		transcript,
	)
}

func MeetingsWithSummaryList(
	meetings []models.Meeting, total int, dateFormat string, summaryLength int,
) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 Встречи (всего %d):\n\n", total))
	for i, m := range meetings {
		sb.WriteString(fmt.Sprintf("%d. ID: %d\n", i+1, m.ID))
		sb.WriteString(fmt.Sprintf("📅 Дата: %s\n", m.CreatedAt.Format(dateFormat)))
		if m.IsTranscriptionFailed {
			sb.WriteString("⚠️ Возникли технические неполадки. Резюме не будет создано.\n\n")
		} else if m.Summary == nil {
			sb.WriteString(fmt.Sprintf("📄 Резюме:\n%s\n\n", "⏳ Ещё создаю. Осталось совсем немного."))
		} else {
			sb.WriteString(fmt.Sprintf("📄 Резюме:\n%s\n\n", truncate(*m.Summary, summaryLength)))
		}
	}

	sb.WriteString("💡 Используй `/get <id встречи>` чтобы увидеть транскрипцию.\n")
	if total > len(meetings) {
		sb.WriteString("Используй `/list <№ страницы>` чтобы увидеть другие встречи.")
	}

	return sb.String()
}

func truncate(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}

	runes := []rune(s)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes]) + "..."
	}

	return s
}
