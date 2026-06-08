package paidsub

import "strings"

// Minimal server-side i18n catalog for the bot (the Vue catalog is frontend
// only). Language is chosen from the Telegram user's language_code, defaulting
// to English; Russian is provided since the panel defaults to a Moscow tz.

type lang string

const (
	langEN lang = "en"
	langRU lang = "ru"
)

func pickLang(languageCode string) lang {
	if strings.HasPrefix(strings.ToLower(languageCode), "ru") {
		return langRU
	}
	return langEN
}

var messages = map[lang]map[string]string{
	langEN: {
		"greeting":              "👋 Welcome! Use the buttons below to get your subscription, QR codes and usage stats.",
		"not_linked":            "Your Telegram is not linked to an account. Please contact the administrator.",
		"menu_links":            "🔗 My links",
		"menu_qr":               "🔳 QR codes",
		"menu_stats":            "👤 My subscription",
		"menu_payment":          "💳 Payment",
		"menu_buy":              "💳 Buy / Renew",
		"menu_orders":           "🧾 My purchases",
		"menu_refund":           "↩️ Request a refund",
		"menu_back":             "⬅️ Back",
		"menu_help":             "❓ Help",
		"help":                  "Use the buttons to get your subscription links, QR codes and current usage. Send /start to open the menu.",
		"payment_title":         "💳 Payment — choose an option:",
		"links_title":           "Your subscription:",
		"links_none":            "No links available yet. Please contact the administrator.",
		"qr_caption_sub":        "Subscription QR",
		"stats_title":           "👤 Your subscription",
		"stats_used":            "Used",
		"stats_limit":           "Limit",
		"stats_unlim":           "unlimited",
		"stats_expiry":          "Expires in",
		"stats_days":            "days",
		"stats_expired":         "expired",
		"stats_online":          "Online",
		"stats_offline":         "Offline",
		"stats_enabled":         "Active",
		"stats_disabled":        "Disabled",
		"rate_limited":          "Too many requests, please wait a moment.",
		"error":                 "Something went wrong, please try again later.",
		"registered":            "✅ A trial subscription has been created for you! Use the buttons below.",
		"reg_full":              "Registration is temporarily unavailable. Please try again later.",
		"reg_no_setup":          "Self-registration is not configured. Please contact the administrator.",
		"buy_title":             "Choose a plan:",
		"buy_none":              "No plans are available right now. Please contact the administrator.",
		"buy_choose_provider":   "Choose a payment method:",
		"pay_open":              "Open payment",
		"pay_open_hint":         "Tap the button below to pay. Your subscription updates automatically after payment.",
		"pay_manual_btn":        "✅ I have paid",
		"pay_manual_sent":       "Thanks! We will verify your payment and update your subscription shortly.",
		"pay_invoice_failed":    "Could not create the invoice. Please try again later.",
		"pay_success":           "✅ Payment received — your subscription has been updated!",
		"orders_title":          "🧾 Your purchases:",
		"orders_none":           "You have no purchases yet.",
		"order_status_pending":  "⏳ pending",
		"order_status_paid":     "✅ paid",
		"order_status_failed":   "❌ failed",
		"order_status_expired":  "⌛ expired",
		"order_status_canceled": "🚫 canceled",
		"order_status_refunded": "↩️ refunded",
		"refund_choose":         "Choose a purchase to refund:",
		"refund_none":           "You have no purchases available for refund.",
		"refund_not_eligible":   "This purchase can no longer be refunded.",
		"refund_done":           "✅ Your payment has been refunded.",
		"refund_requested":      "Your refund request has been sent to the administrator.",
	},
	langRU: {
		"greeting":              "👋 Добро пожаловать! Используйте кнопки ниже, чтобы получить подписку, QR-коды и статистику.",
		"not_linked":            "Ваш Telegram не привязан к аккаунту. Обратитесь к администратору.",
		"menu_links":            "🔗 Мои ссылки",
		"menu_qr":               "🔳 QR-коды",
		"menu_stats":            "👤 Моя подписка",
		"menu_payment":          "💳 Оплата",
		"menu_buy":              "💳 Купить / Продлить",
		"menu_orders":           "🧾 Мои покупки",
		"menu_refund":           "↩️ Оформить возврат",
		"menu_back":             "⬅️ Назад",
		"menu_help":             "❓ Помощь",
		"help":                  "Используйте кнопки, чтобы получить ссылки подписки, QR-коды и текущее потребление. Отправьте /start, чтобы открыть меню.",
		"payment_title":         "💳 Оплата — выберите действие:",
		"links_title":           "Ваша подписка:",
		"links_none":            "Ссылки пока недоступны. Обратитесь к администратору.",
		"qr_caption_sub":        "QR подписки",
		"stats_title":           "👤 Ваша подписка",
		"stats_used":            "Использовано",
		"stats_limit":           "Лимит",
		"stats_unlim":           "безлимит",
		"stats_expiry":          "Осталось",
		"stats_days":            "дней",
		"stats_expired":         "истекла",
		"stats_online":          "Онлайн",
		"stats_offline":         "Не в сети",
		"stats_enabled":         "Активна",
		"stats_disabled":        "Отключена",
		"rate_limited":          "Слишком много запросов, подождите немного.",
		"error":                 "Что-то пошло не так, попробуйте позже.",
		"registered":            "✅ Для вас создана пробная подписка! Используйте кнопки ниже.",
		"reg_full":              "Регистрация временно недоступна. Попробуйте позже.",
		"reg_no_setup":          "Саморегистрация не настроена. Обратитесь к администратору.",
		"buy_title":             "Выберите тариф:",
		"buy_none":              "Сейчас нет доступных тарифов. Обратитесь к администратору.",
		"buy_choose_provider":   "Выберите способ оплаты:",
		"pay_open":              "Перейти к оплате",
		"pay_open_hint":         "Нажмите кнопку ниже для оплаты. После оплаты подписка обновится автоматически.",
		"pay_manual_btn":        "✅ Я оплатил",
		"pay_manual_sent":       "Спасибо! Мы проверим оплату и скоро обновим вашу подписку.",
		"pay_invoice_failed":    "Не удалось создать счёт. Попробуйте позже.",
		"pay_success":           "✅ Оплата получена — ваша подписка обновлена!",
		"orders_title":          "🧾 Ваши покупки:",
		"orders_none":           "У вас пока нет покупок.",
		"order_status_pending":  "⏳ ожидает",
		"order_status_paid":     "✅ оплачен",
		"order_status_failed":   "❌ ошибка",
		"order_status_expired":  "⌛ истёк",
		"order_status_canceled": "🚫 отменён",
		"order_status_refunded": "↩️ возвращён",
		"refund_choose":         "Выберите покупку для возврата:",
		"refund_none":           "Нет покупок, доступных для возврата.",
		"refund_not_eligible":   "Эту покупку больше нельзя вернуть.",
		"refund_done":           "✅ Платёж возвращён.",
		"refund_requested":      "Заявка на возврат отправлена администратору.",
	},
}

func tr(l lang, key string) string {
	if m, ok := messages[l]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	if v, ok := messages[langEN][key]; ok {
		return v
	}
	return key
}
