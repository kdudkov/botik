package alert

type NotificationType int

// 2. Используем const и iota для объявления набора констант
const (
	NotificationBad NotificationType = iota
	NotificationGood
	NotificationRemind
)

type Notification struct {
	alert *Alert
	typ   NotificationType
}
