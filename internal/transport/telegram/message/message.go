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
• /list - покажу все твои встречи
• /get <id> - покажу транскрипцию встречи
• /find <слово/слова> - найду встречи по ключевым словам и покажу
• /chat <вопрос> - отвечу на вопрос по встречам
• /help - покажу список дотупных комманд и возможностей

🎙️ Отправь голосовое сообщение или аудио файл - сделаю транскрипцию встречи и резюме
💡 Напиши вопрос без команды для продолжения последнего диалога
`

const WelcomeBack = `
👋 Рад снова видеть тебя, %s!

🤖 Чем могу помочь?

🕹️ Введи команду:
• /list - покажу все твои встречи
• /get <id встречи> - покажу транскрипцию встречи
• /find <слово/слова> - найду встречи по ключевым словам и покажу
• /chat <вопрос> - отвечу на вопрос по встречам
• /help - покажу список дотупных комманд и возможностей

🎙️ Отправь голосовое сообщение или аудио файл - сделаю транскрипцию встречи и резюме
💡 Напиши вопрос без команды для продолжения последнего диалога
`

const Help = `
🕹️ команды:
• /start - зарегистрироваться 
• /list - показать список всех встреч
• /get <id встречи> - показать транскрипцию встречи
• /find <слово/слова> - найти встречи по ключевым словам
• /chat <вопрос> - получить ответ на вопрос по встречам
• /help - показать это сообщение

🎙️ аудио/голосовое - сделать транскрипцию встречи и резюме
💡 сообщение без команды - продолжение последнего диалога
`

const IdentificationFailed = `
🚫 Разработчик запретил мне общаться с незнакомцами.
Используй команду /start чтобы подружиться.
`

const NoMeetings = `
📭 Встреч пока нет.
Отправь аудио файл или голосовое сообщение, чтобы начать.
`

const EmptyOrTooMuchMeetingID = `
🔢 Укажи ID одной встречи.
Например: /get 123
`

const IncorrectMeetingID = `
🔢 Укажи корректный ID встречи.
Например: /get 123
`

const MeetingNotFound = `
🔍 Нет такой встречи.
Используй команду /list чтобы увидеть список всех твоих встреч.
`

const EmptyFindQuery = `
💬 Напиши ключевое слово/слова для поиска.
Например: /find добрый день
`

const MeetingsNotFound = `
📭 Ничего не нашел.
Попробуй поискать по другим ключевым словам.
`

const OperationFailed = `
⚠️ Произошла непредвиденная ошибка.
Попробуй повторить операцию.
`

const OperationTimeout = `
🐌 Похоже, у меня слишком много дел, и я не успеваю.
Попробуй повторить операцию позднее.
`

func MeeetingWithTranscript(meeting models.MeetingWithTranscript, dateFormat string, transcriptLength int) string {
	return fmt.Sprintf(
		"🗓️ *Встреча:* `%d`\n📅 *Дата:* %s\n\n*📜 Транскрипция:*\n%s",
		meeting.ID,
		meeting.CreatedAt.Format(dateFormat),
		truncate(meeting.Transcript, transcriptLength),
	)
}

func MeetingsWithSummaryList(meetings []models.MeetingWithSummary, dateFormat string, summaryLength int) string {
	var sb strings.Builder
	sb.WriteString("📋 *Встречи:*\n\n")
	for i, m := range meetings {
		sb.WriteString(fmt.Sprintf("*%d.* `ID: %d`\n", i+1, m.ID))
		sb.WriteString(fmt.Sprintf("📅 Дата: %s\n", m.CreatedAt.Format(dateFormat)))
		if m.Summary == "" {
			sb.WriteString("⏳ Ещё составляю резюме. Осталось совсем немного")
		} else {
			sb.WriteString(fmt.Sprintf("📄 Резюме: %s\n\n", truncate(m.Summary, summaryLength)))
		}
	}
	sb.WriteString("💡 Используй `/get <id>` чтобы увидеть транскрипцию встречи")

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
