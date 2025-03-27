package callback

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/api"
	"github.com/hellodex/tradingbot/entity"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
)

func SettingHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	isDisabled := true

	chatID := util.EffectId(update)
	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			// line 1
			{
				entity.GetCallbackButton(entity.SETTING_SLIPPY),
			},

			// line2
			{},
		},
	}

	userInfo, err := api.GetUserProfile(chatID)
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatID, "出问题了，请联系管理员")
		return
	}
	textTempl := `
请选择你要设置的内容：
当前设置：
滑点：%s%%
<code>UUID: %s</code>
	`

	store.BotMessageAdd()
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        fmt.Sprintf(textTempl, userInfo.FromPercentage(userInfo.Data.Slippage), userInfo.Data.UUID),
		ParseMode:   "HTML",
		ReplyMarkup: kb,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: &isDisabled,
		},
	})
}

var ReplaySlippySettingCache = make(map[int64]int)

func SlippyHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	chatID := update.CallbackQuery.Message.Message.Chat.ID
	reply := models.ForceReply{
		ForceReply:            true,
		InputFieldPlaceholder: "20",
	}

	store.BotMessageAdd()
	message, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        "请输入数字设置滑点,例如 20 就是设置为20%",
		ReplyMarkup: reply,
	})
	if err != nil {
		log.Error().Err(err).Msg("发送滑点设置消息失败")
		return
	}

	ReplaySlippySettingCache[chatID] = message.ID
}

func HandleSlippyReply(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.ReplyToMessage == nil {
		return
	}

	chatID := update.Message.Chat.ID
	replyToID := update.Message.ReplyToMessage.ID

	// 检查是否是对滑点设置的回复
	if cachedMsgID, exists := ReplaySlippySettingCache[chatID]; !exists || cachedMsgID != replyToID {
		return
	}

	// 解析用户输入的滑点值
	slippageNum, err := cast.ToFloat64E(update.Message.Text)
	if err != nil {
		store.BotMessageAdd()
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ 请输入有效的数字",
		})
		return
	}

	// 验证滑点值范围
	if slippageNum < 0 || slippageNum > 100 {
		store.BotMessageAdd()
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ 滑点值必须在 0-100 之间",
		})
		return
	}

	// 清理会话缓存
	delete(ReplaySlippySettingCache, chatID)

	userInfo, err := api.GetUserProfile(chatID)
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatID, "出问题了，请联系管理员")
		return
	}
	userInfo.Data.Slippage = userInfo.ToPercentage(slippageNum)
	fmt.Println(userInfo.Data.Slippage)
	err = api.UpdateUserProfile(chatID, userInfo)
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatID, err.Error())
		return
	}

	// 发送设置成功消息
	store.BotMessageAdd()
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   fmt.Sprintf("✅ 滑点设置成功：%.2f%%", slippageNum),
	})
}
