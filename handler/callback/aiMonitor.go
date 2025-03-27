package callback

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/api"
	"github.com/hellodex/tradingbot/logger"
	"github.com/hellodex/tradingbot/model"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/template"
	"github.com/samber/lo"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"

	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog/log"
)

var aiMonitorMenuTempl = `
HelloDex: AI 监控

可监控推特,钱包,代币等信息
通过TG机器人,网页,APP提醒
`

// handle callback button ai monitor button
func CallbackAIMonitorMenu(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)

	kb := util.NewAiMonitorKeyboard()
	store.BotMessageAdd()
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		Text:        aiMonitorMenuTempl,
		ParseMode:   "HTML",
		ReplyMarkup: kb,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
	if err != nil {
		log.Error().Err(err).Send()
	}
}

var aiMonitorTextTempl = `
HelloDex: Ai监控

%s
当前价格: $%s
目标价格: $%s
`

// handle ai monitor list
func CallbackAIMonitorList(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)

	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	dw := func() *model.Wallet {
		for _, wallets := range userInfo.Data.Wallets {
			for _, w := range wallets {
				if w.WalletId == userInfo.Data.TgDefaultWalletId {
					return &w
				}
			}
		}
		return &model.Wallet{}
	}()

	if dw == nil {
		log.Error().Msg("get default wallet error")
		return
	}

	data, err := api.ListUserTokenSubscribe("SOLANA", userInfo)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	listData := gjson.GetBytes(data.([]byte), "data.subscribeList").Raw
	message, err := template.RanderListAimonitor([]byte(listData))
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	store.BotMessageAdd()
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatId,
		Text:      message,
		ParseMode: "HTML",
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
	if err != nil {
		log.Error().Err(err).Send()
	}
}

// callback for select ai monitor type
func BuildAiMonitorSelectTypeKeyboard(chatId any) (models.InlineKeyboardMarkup, error) {
	var buttons [][]models.InlineKeyboardButton

	prefix := fmt.Sprintf("monitor_select::%v", chatId)

	// First row of buttons
	priceMonitorButton := models.InlineKeyboardButton{
		Text:         "价格监控",
		CallbackData: prefix + "::price",
	}

	volMonitorButton := models.InlineKeyboardButton{
		Text:         "波跌幅监控",
		CallbackData: prefix + "::chg",
	}

	// Second row of buttons
	largeOrderBuyButton := models.InlineKeyboardButton{
		Text:         "大单买入监控",
		CallbackData: prefix + "::buy",
	}

	largeOrderSellButton := models.InlineKeyboardButton{
		Text:         "大单卖出监控",
		CallbackData: prefix + "::sell",
	}

	// Add rows to buttons
	buttons = append(buttons, []models.InlineKeyboardButton{priceMonitorButton, volMonitorButton})
	buttons = append(buttons, []models.InlineKeyboardButton{largeOrderBuyButton, largeOrderSellButton})

	// Create and return the keyboard markup
	keyboard := models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	return keyboard, nil
}

// callback: select chain for ai monitor
func BuildChainsMenuSelectForAImonitor(chatID any, userInfo model.GetUserResp) (models.InlineKeyboardMarkup, error) {
	var buttons [][]models.InlineKeyboardButton

	chainCfgs, err := api.GetChainConfigs()
	if err != nil {
		return models.InlineKeyboardMarkup{}, err
	}

	slices.SortFunc(chainCfgs.Data, func(a, b model.ChainConfig) int {
		if a.Sort < b.Sort {
			return -1
		}
		if a.Sort > b.Sort {
			return 1
		}
		return 0
	})

	for _, chainCfg := range chainCfgs.Data {
		displayText := chainCfg.Chain

		dw, _, _ := UserDefaultWalletInfo(userInfo)
		if dw.ChainCode == chainCfg.ChainCode {
			displayText = "✅" + displayText
		}

		callbackData := fmt.Sprintf("aiSchain::%v::%s", chatID, chainCfg.ChainCode)

		button := models.InlineKeyboardButton{
			Text:         displayText,
			CallbackData: callbackData,
		}
		buttons = append(buttons, []models.InlineKeyboardButton{button})
	}

	return models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}, nil
}

// handle callback confirm select
// callback prefix: aiSchain
// aiSchain::tgid::chainCode
func CallbackUserConfirmChainForAimonitor(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)
	callback := u.CallbackQuery
	params := strings.Split(callback.Data, "::")
	log.Debug().Interface("callback params: ", params).Send()

	dateByte, has := store.UserGetAiMonitorInfo(chatId)
	if !has {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}

	reqData := &model.AISubscribeReqData{}
	err := json.Unmarshal(dateByte, reqData)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}
	reqData.ChainCode = params[len(params)-1]

	if data, err := reqData.JsonB(); err == nil {
		err := store.UserSetAiMonitorInfo(chatId, data)
		if err != nil {
			log.Error().Err(err).Send()
			return
		}
	}
	reply := models.ForceReply{
		ForceReply:            true,
		InputFieldPlaceholder: "请输入代币合约地址",
	}

	store.BotMessageAdd()
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatId,
		Text:      "请输入代币合约地址",
		ParseMode: "HTML",
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
		ReplyMarkup: reply,
	})
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
}

// select type for monitor type
func CallabckAiMonitorSelectType(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)

	kb, err := BuildAiMonitorSelectTypeKeyboard(chatId)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	message := `
HelloDex: AI监控

请选择监控类型
	`

	store.BotMessageAdd()
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		Text:        message,
		ParseMode:   "HTML",
		ReplyMarkup: kb,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
	if err != nil {
		log.Error().Err(err).Send()
	}
}

// monitor_select::chatId::type
func CallbackAimonitorConfirmType(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)
	callback := u.CallbackQuery
	params := strings.Split(callback.Data, "::")
	log.Debug().Interface("callback params: ", params).Send()

	reqData := &model.AISubscribeReqData{
		MonitorType: params[len(params)-1],
	}
	if data, err := reqData.JsonB(); err == nil {
		err := store.UserSetAiMonitorInfo(chatId, data)
		if err != nil {
			log.Error().Err(err).Send()
			return
		}
	}

	CallbackAiMonitorSelectChain(ctx, b, u)
}

// callback select chain for aimonitor
func CallbackAiMonitorSelectChain(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)

	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	kb, err := BuildChainsMenuSelectForAImonitor(chatId, userInfo)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	message := `
HelloDex: AI监控

请选择需要监控的公链
	`

	store.BotMessageAdd()
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		Text:        message,
		ParseMode:   "HTML",
		ReplyMarkup: kb,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
	if err != nil {
		log.Error().Err(err).Send()
	}
}

func CallbackAiMonitorTokenInfoSetting(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)

	kb := util.AiMonitorSettingsKeyBoard(util.SettingsKeyBoardData{
		EnableTG:  false,
		EnableWeb: false,
		EnableApp: false,
		Frequency: "one week",
	})
	store.BotMessageAdd()
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		Text:        fmt.Sprintf(aiMonitorTextTempl, "Symbol", "100", "100"),
		ParseMode:   "HTML",
		ReplyMarkup: kb,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
	if err != nil {
		log.Error().Err(err).Send()
	}
}

// button("添加代币推送", "add_token_alert"),
// button("取消设置", "cancel_settings"),
func Callback_add_token_alert(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)
	datad, has := store.UserGetAiMonitorInfo(chatId)
	if !has {
		return
	}

	var subReq model.AISubscribeReqData
	err := json.Unmarshal(datad, &subReq)
	if err != nil {
		return
	}

	if subReq.Vaild() {
		userInfo, err := api.GetUserProfile(chatId)
		if err != nil {
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			return
		}
		_, err = api.UpdateCommonSubscribe(subReq, userInfo)
		if err != nil {
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			return
		}

		if err == nil {
			// util.QuickMessage(ctx, b, chatId, "添加监听成功")
			util.QuickMessageWithButton(ctx, b, chatId, "添加监听成功", util.BackToMainMenu())
			// set notify
			// prefix: toggle_TG/APP/网页

			subList := userInfo.Data.SubscribeSetting
			if !lo.Contains(subList, "telegram") {
				subList = append(subList, "telegram")
			}

			logger.StdLogger().Info().Interface("sub list", subList).Send()
			_, err = api.UpdateUserSubscribeSetting(subList, userInfo)
			if err != nil {
				util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
				return
			}
			b.DeleteMessages(ctx, &bot.DeleteMessagesParams{
				ChatID:     chatId,
				MessageIDs: []int{subReq.SessionMessageID},
			})

		}
	} else {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}
	store.UserDeleteAiMonitorInfo(chatId)
}

// callback for cancel button
func Callback_cancel_settings(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)
	datad, has := store.UserGetAiMonitorInfo(chatId)
	if !has {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}

	var subReq model.AISubscribeReqData
	err := json.Unmarshal(datad, &subReq)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}

	util.QuickMessageWithButton(ctx, b, chatId, "已经取消监听设置", util.BackToMainMenu())
	b.DeleteMessages(ctx, &bot.DeleteMessagesParams{
		ChatID:     chatId,
		MessageIDs: []int{subReq.SessionMessageID},
	})
	store.UserDeleteAiMonitorInfo(chatId)
}

func toggleSlice(old, new []string) []string {
	result := make([]string, len(old))
	copy(result, old)

	for _, item := range new {
		if lo.Contains(result, item) {
			result = lo.Filter(result, func(s string, _ int) bool {
				return s != item
			})
		} else {
			result = append(result, item)
		}
	}

	return result
}

func localUpdateUserProfileChannelList(chatId int64, list []string) {
	var result model.GetUserResp
	data, ok := store.RedisGetUserProfile(chatId)
	if !ok {
		log.Error().Msg("can't find user profile in redis")
		return
	}

	err := json.Unmarshal(data, &result)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	result.Data.SubscribeSetting = list

	if ok := store.RedisSetUserProfile(chatId, result); !ok {
		log.Debug().
			Int64("user_id", chatId).
			Msg("Failed to set user profile in Redis")
	}
}

// status keyboard for aimonitor info
func newKeyboardToggle(newStatus []string, u *models.Update) models.InlineKeyboardMarkup {
	kb := u.CallbackQuery.Message.Message.ReplyMarkup
	for i, buttons := range kb.InlineKeyboard {
		for j, b := range buttons {
			switch b.CallbackData {
			case "toggle_TG":
				kb.InlineKeyboard[i][j] = util.GetToggleButton("TG", lo.Contains(newStatus, "telegram"))
			case "toggle_网页":
				kb.InlineKeyboard[i][j] = util.GetToggleButton("网页", lo.Contains(newStatus, "web"))
			case "toggle_APP":
				kb.InlineKeyboard[i][j] = util.GetToggleButton("APP", lo.Contains(newStatus, "app"))
			}
		}
	}
	return kb
}

// prefix: toggle_TG/APP/网页
func CallbackToggleChannels(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)
	callbackData := u.CallbackQuery.Data
	log.Debug().Interface("callbackData", callbackData).Send()
	params := strings.Split(callbackData, "_")
	profile, err := api.GetUserProfile(chatId)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	old := profile.Data.SubscribeSetting
	log.Debug().Interface("old list", old).Send()
	switch params[1] {
	case "TG":
		result := toggleSlice(old, []string{"telegram"})
		_, err := api.UpdateUserSubscribeSetting(result, profile)
		if err != nil {
			log.Error().Err(err).Send()
			return
		}

		kb := newKeyboardToggle(result, u)
		_, err = b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
			ChatID:      chatId,
			MessageID:   u.CallbackQuery.Message.Message.ID,
			ReplyMarkup: kb,
		})
		if err != nil {
			log.Error().Err(err).Send()
			return
		}
		localUpdateUserProfileChannelList(chatId, result)

	case "网页":
		result := toggleSlice(old, []string{"web"})
		_, err := api.UpdateUserSubscribeSetting(result, profile)
		if err != nil {
			log.Error().Err(err).Send()
			return
		}
		kb := newKeyboardToggle(result, u)
		_, err = b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
			ChatID:      chatId,
			MessageID:   u.CallbackQuery.Message.Message.ID,
			ReplyMarkup: kb,
		})
		if err != nil {
			log.Error().Err(err).Send()
			return
		}
		localUpdateUserProfileChannelList(chatId, result)
	case "APP":
		new := strings.ToLower(params[1])
		result := toggleSlice(old, []string{new})
		_, err := api.UpdateUserSubscribeSetting(result, profile)
		if err != nil {
			log.Error().Err(err).Send()
			return
		}
		kb := newKeyboardToggle(result, u)
		_, err = b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
			ChatID:      chatId,
			MessageID:   u.CallbackQuery.Message.Message.ID,
			ReplyMarkup: kb,
		})
		if err != nil {
			log.Error().Err(err).Send()
			return
		}
		localUpdateUserProfileChannelList(chatId, result)
	default:
	}
}

// check button callbackData
func newKeyboardFreq(newButtonSuffix string, u *models.Update) models.ReplyMarkup {
	kb := u.CallbackQuery.Message.Message.ReplyMarkup
	checkmark := "✅"

	// 更新按钮状态
	for i, buttons := range kb.InlineKeyboard {
		for j, b := range buttons {
			if !strings.HasPrefix(b.CallbackData, "freq_") {
				continue
			}

			buttonText := b.Text
			if strings.HasPrefix(buttonText, checkmark) {
				buttonText = strings.TrimSpace(buttonText[len(checkmark):])
			}

			suffix := strings.TrimPrefix(b.CallbackData, "freq_")
			if suffix == newButtonSuffix {
				buttonText = checkmark + " " + buttonText
			}

			kb.InlineKeyboard[i][j].Text = buttonText
		}
	}

	return kb
}

// prefix: freq_1/2/3
func CallbackUpdateFreqSetting(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)
	callbackData := u.CallbackQuery.Data
	log.Debug().Interface("callbackData", callbackData).Send()
	params := strings.Split(callbackData, "_")
	monitoryInfoData, has := store.UserGetAiMonitorInfo(chatId)
	if !has {
		log.Error().Msg("handler callback update frequency can't get subReq")
		return
	}

	subReq := &model.AISubscribeReqData{}
	err := json.Unmarshal(monitoryInfoData, subReq)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}

	targetType := params[len(params)-1]
	subReq.NoticeType = cast.ToInt(targetType)

	kb := newKeyboardFreq(targetType, u)
	_, err = b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:      chatId,
		MessageID:   u.CallbackQuery.Message.Message.ID,
		ReplyMarkup: kb,
	})
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	newData, err := subReq.JsonB()
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	err = store.UserSetAiMonitorInfo(chatId, newData)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
}

// callback for aimonitor pause
func CallbackHandleAimonitorPuserButton(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)
	callbackData := u.CallbackQuery.Data
	log.Debug().Msg(callbackData)

	messageid := cast.ToString(u.CallbackQuery.Message.Message.ID)

	reqBodyStr, err := store.GetMessageByMsgId(chatId, messageid)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	var reqbodyMap map[string]string

	err = json.Unmarshal([]byte(reqBodyStr), &reqbodyMap)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	action := strings.TrimPrefix(callbackData, "pusher_")
	switch action {
	case "edit":
		editPusherTokenInfo(ctx, b, chatId, reqbodyMap["chainCode"], reqbodyMap["baseAddress"], "price", userInfo)
	case "pause":
		_, err := api.PauseUserTokenSubscribe(reqbodyMap, userInfo)
		if err != nil {
			log.Error().Err(err).Send()
			return
		}
		util.QuickMessage(ctx, b, chatId, "暂停成功")
	case "delete":
		_, err := api.DeleteUserTokenSubscribe(reqbodyMap, userInfo)
		if err != nil {
			log.Error().Err(err).Send()
			return
		}
		util.QuickMessage(ctx, b, chatId, "删除成功")
	}
}

// callback edit for push info
func editPusherTokenInfo(ctx context.Context, b *bot.Bot, chatId int64, chainCode string, baseAddress string, monitorType string, userInfo model.GetUserResp) {
	data, _, err := api.GetUserTokenSubscribe(chatId, chainCode, baseAddress, monitorType, userInfo)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	freq_Map := map[int64]string{
		1: "once",
		2: "daily",
		3: "every",
	}

	byteStr := gjson.GetBytes(data, "data.subscribe").Raw

	var subTokenInfo model.TokenSubscribeInfo
	err = json.Unmarshal([]byte(byteStr), &subTokenInfo)
	if err != nil {
		log.Error().Err(err).Send()
	}
	keyBoardData := util.SettingsKeyBoardData{
		EnableTG:  lo.Contains(userInfo.Data.SubscribeSetting, "telegram"),
		EnableWeb: lo.Contains(userInfo.Data.SubscribeSetting, "web"),
		EnableApp: lo.Contains(userInfo.Data.SubscribeSetting, "app"),
		Frequency: freq_Map[subTokenInfo.NoticeType],
	}
	kb := util.AiMonitorSettingsKeyBoard(keyBoardData)

	tmplll := `
HelloDex: AI监控
当前价格: $%s
目标价格: $%s
		`

	subReq := model.AISubscribeReqData{
		UserId:       chatId,
		CurrentPrice: subTokenInfo.StartPrice,
		ChainCode:    chainCode,
		BaseAddress:  baseAddress,
		Symbol:       subTokenInfo.Symbol,
		NoticeType:   int(subTokenInfo.NoticeType),
		MonitorType:  monitorType,
		TargetPrice:  subTokenInfo.TargetPrice,
	}
	newData, err := subReq.JsonB()
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}
	store.UserSetAiMonitorInfo(chatId, newData)
	currentPrice := util.FormatNumber(subTokenInfo.StartPrice)
	targetPrice := util.FormatNumber(subTokenInfo.TargetPrice)
	store.BotMessageAdd()
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		Text:        fmt.Sprintf(tmplll, currentPrice, targetPrice),
		ReplyMarkup: kb,
	})
}

// this will be replay by type, and set state in redis to with user to replay
func replayByTypeAndInstate(monitorType string) (models.ForceReply, string) {
	placeholder := ""
	switch monitorType {
	case "price":
		placeholder = "请输入目标价格"
	case "chg":
		placeholder = "请输入目标涨幅"
	case "buy":
		placeholder = "请输入买入交易额"
	case "sell":
		placeholder = "请输入卖出交易额"
	default:
		placeholder = "请输入数额"
	}

	return models.ForceReply{
		ForceReply:            true,
		InputFieldPlaceholder: placeholder,
	}, placeholder
}

// "edit_current_"+monitorType
// "pause_current_"+monitorType
// "delete_current_"+monitorType
func CallBackEditCurrentAimonitor(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)
	callabckData := u.CallbackQuery.Data
	monitorType := strings.TrimPrefix(callabckData, "edit_current_")
	reply, text := replayByTypeAndInstate(monitorType)
	store.BotMessageAdd()
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		Text:        text,
		ReplyMarkup: reply,
	})
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s, 编辑监控", util.AdminUrl))
		log.Error().Err(err).Send()
		return
	}

	store.RedisSetState(chatId, "edit_current", monitorType, 30*time.Second)
}

// callback delete currnet aimonitor
func CallBackDeleteCurrentAimonitor(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)
	defer func() {
		store.RedisDeleteState(chatId, "edit_current")
		store.UserDeleteAiMonitorInfo(chatId)
	}()
	dataB, has := store.UserGetAiMonitorInfo(chatId)
	if !has {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s, 删除监控", util.AdminUrl))
		log.Error().Msg("cant get user aimonitorinfo")
		return
	}

	var reqDataRaw model.AISubscribeReqData
	err := json.Unmarshal(dataB, &reqDataRaw)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s, 删除监控", util.AdminUrl))
		log.Error().Err(err).Send()
		return
	}

	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s, 删除监控", util.AdminUrl))
		log.Error().Err(err).Send()
		return
	}

	reqData := map[string]string{
		"chainCode":   reqDataRaw.ChainCode,
		"baseAddress": reqDataRaw.BaseAddress,
		"type":        reqDataRaw.MonitorType,
	}
	_, err = api.DeleteUserTokenSubscribe(reqData, userInfo)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s, 删除监控", util.AdminUrl))
		log.Error().Err(err).Send()
		return
	}

	util.QuickMessageWithButton(ctx, b, chatId, "删除成功", util.BackToMainMenu())
	b.DeleteMessages(ctx, &bot.DeleteMessagesParams{
		ChatID:     chatId,
		MessageIDs: []int{reqDataRaw.SessionMessageID},
	})
}

// callback pause currnet aimonitor
func CallBackPauseCurrentAimonitor(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)
	defer func() {
		store.RedisDeleteState(chatId, "edit_current")
		store.UserDeleteAiMonitorInfo(chatId)
	}()
	dataB, has := store.UserGetAiMonitorInfo(chatId)
	if !has {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s, 暂停监控", util.AdminUrl))
		log.Error().Msg("cant get user aimonitorinfo")
		return
	}

	var reqDataRaw model.AISubscribeReqData
	err := json.Unmarshal(dataB, &reqDataRaw)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s, 暂停监控", util.AdminUrl))
		log.Error().Err(err).Send()
		return
	}

	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s, 暂停监控", util.AdminUrl))
		log.Error().Err(err).Send()
		return
	}

	reqData := map[string]string{
		"chainCode":   reqDataRaw.ChainCode,
		"baseAddress": reqDataRaw.BaseAddress,
		"type":        reqDataRaw.MonitorType,
	}
	_, err = api.PauseUserTokenSubscribe(reqData, userInfo)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s, 暂停监控", util.AdminUrl))
		log.Error().Err(err).Send()
		return
	}

	util.QuickMessageWithButton(ctx, b, chatId, "暂停成功", util.BackToMainMenu())
	b.DeleteMessages(ctx, &bot.DeleteMessagesParams{
		ChatID:     chatId,
		MessageIDs: []int{reqDataRaw.SessionMessageID},
	})
}

// callback save currnet aimonitor
func CallbackSaveCurrentAimonitor(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)
	defer func() {
		store.RedisDeleteState(chatId, "edit_current")
		store.UserDeleteAiMonitorInfo(chatId)
	}()
	dataB, has := store.UserGetAiMonitorInfo(chatId)
	if !has {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s, 保存监控", util.AdminUrl))
		log.Error().Msg("cant get user aimonitorinfo")
		return
	}

	var reqData model.AISubscribeReqData
	err := json.Unmarshal(dataB, &reqData)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s, 保存监控", util.AdminUrl))
		log.Error().Err(err).Send()
		return
	}

	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s, 保存监控", util.AdminUrl))
		log.Error().Err(err).Send()
		return
	}

	_, err = api.UpdateCommonSubscribe(reqData, userInfo)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s, 保存监控", util.AdminUrl))
		log.Error().Err(err).Send()
		return
	}

	util.QuickMessageWithButton(ctx, b, chatId, "保存成功", util.BackToMainMenu())
	b.DeleteMessages(ctx, &bot.DeleteMessagesParams{
		ChatID:     chatId,
		MessageIDs: []int{reqData.SessionMessageID},
	})
}

func CallbackPusherQuickTradeButton(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)
	callbackData := u.CallbackQuery.Data
	log.Debug().Msg(callbackData)

	messageid := cast.ToString(u.CallbackQuery.Message.Message.ID)

	reqBodyStr, err := store.GetMessageByMsgId(chatId, messageid)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	var reqbodyMap map[string]string

	err = json.Unmarshal([]byte(reqBodyStr), &reqbodyMap)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	newUpdate := &models.Update{
		Message: &models.Message{
			Text: reqbodyMap["baseAddress"],
			From: &u.CallbackQuery.From,
			Chat: u.CallbackQuery.Message.Message.Chat,
			ID:   u.CallbackQuery.Message.Message.ID,
			Date: int(time.Now().Unix()),
		},
		ID: u.ID,
	}

	b.ProcessUpdate(ctx, newUpdate)
}

func ListAimonitorWithButton(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)

	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	dwChainCode := ""
	for _, wallets := range userInfo.Data.Wallets {
		for _, w := range wallets {
			if w.WalletId == userInfo.Data.TgDefaultWalletId {
				dwChainCode = w.ChainCode
			}
		}
	}

	data, err := api.ListUserTokenSubscribe(dwChainCode, userInfo)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	subList := gjson.GetBytes(data.([]byte), "data.subscribeList")
	if !subList.Exists() || subList.Array() == nil || len(subList.Array()) == 0 {
		// List is empty or doesn't exist

		CallBackNoSublistAction(ctx, b, u)
		return
	}

	getNotifyType := func(nt int64) string {
		switch nt {
		case 0:
			return "关闭"
		case 1:
			return "1次"
		case 2:
			return "1日1次"
		case 3:
			return "每次"
		}
		return ""
	}

	getMonitorType := func(mt string) string {
		switch mt {
		case "price":
			return "价格"
		case "chg":
			return "涨跌幅"
		case "buy":
			return "买入交易额"
		case "sell":
			return "卖出交易额"
		}
		return ""
	}

	kb := models.InlineKeyboardMarkup{}
	subList.ForEach(func(key, value gjson.Result) bool {
		dataInner := value.Get("data").String()
		baseAddress := value.Get("baseAddress").String()
		text := ""

		if dataInner != "" {
			symbol := value.Get("symbol").String()
			nt := getNotifyType(value.Get("noticeType").Int())
			monitorType := getMonitorType(value.Get("type").String())

			if value.Get("type").String() == "chg" {
				text = fmt.Sprintf("%s-%s-%s: %s", symbol, nt, monitorType, util.FormatPercentage(dataInner))
			} else {
				text = fmt.Sprintf("%s-%s-%s: %s", symbol, nt, monitorType, dataInner)
			}
		} else {
			symbol := value.Get("symbol").String()
			nt := getNotifyType(value.Get("noticeType").Int())
			monitorType := getMonitorType(value.Get("type").String())
			price := util.FormatNumber(value.Get("targetPrice").String())
			logger.StdLogger().Debug().Str("price", price).Msg("")

			text = fmt.Sprintf("%s-%s-%s: $%s", symbol, nt, monitorType, price)
		}

		kb.InlineKeyboard = append(kb.InlineKeyboard, []models.InlineKeyboardButton{
			{Text: text, CallbackData: "aiL:" + baseAddress},
		})
		return true
	})
	store.BotMessageAdd()
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		Text:        "点击监控即可编辑",
		ReplyMarkup: kb,
	})
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	store.RedisSetUserAiList(chatId, subList.Raw)
}

func CallBackNoSublistAction(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatId := util.EffectId(update)
	text := "请先添加监控"
	kb := models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				models.InlineKeyboardButton{Text: "添加监控", CallbackData: "ai_add_monitor"},
				util.BackToMainMenu(),
			},
		},
	}
	store.BotMessageAdd()
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		Text:        text,
		ReplyMarkup: kb,
	})
}

func CallbackHendlerSelectEditAimonitor(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatId := util.EffectId(update)

	callbackData := update.CallbackQuery.Data
	baseAddress := strings.TrimPrefix(callbackData, "aiL:")
	dataB, err := store.RedisGetUserAiList(chatId)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	jsonStr := gjson.ParseBytes(dataB)

	var pick gjson.Result
	jsonStr.ForEach(func(key, value gjson.Result) bool {
		if value.Get("baseAddress").String() == baseAddress {
			pick = value
			return false
		}
		return true
	})
	result := []byte(pick.Raw)

	monitorType := gjson.GetBytes(result, "type").String()
	freq_Map := map[int64]string{
		1: "once",
		2: "daily",
		3: "every",
	}

	var subTokenInfo model.TokenSubscribeInfo
	err = json.Unmarshal(result, &subTokenInfo)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	keyBoardData := util.SettingsKeyBoardData{
		EnableTG:  lo.Contains(userInfo.Data.SubscribeSetting, "telegram"),
		EnableWeb: lo.Contains(userInfo.Data.SubscribeSetting, "web"),
		EnableApp: lo.Contains(userInfo.Data.SubscribeSetting, "app"),
		Frequency: freq_Map[subTokenInfo.NoticeType],
	}
	kb := util.AiMonitor_EditSettingsKeyBoard(keyBoardData, monitorType, int(subTokenInfo.NoticeType))
	tmplll := func() string {
		switch monitorType {
		case "price":
			return `
HelloDex: AI监控
当前价格: $%s
目标价格: $%s
						`
		case "chg":
			return `
HelloDex: AI监控
目标涨跌幅: %s
						`
		case "buy":
			return `
HelloDex: AI监控
买入交易额: %s
						`
		case "sell":
			return `
HelloDex: AI监控
卖出交易额: %s
						`
		}
		return ""
	}()

	var subReq model.AISubscribeReqData
	subReq.MonitorType = monitorType
	subReq.UserId = chatId
	subReq.CurrentPrice = subTokenInfo.StartPrice
	subReq.ChainCode = subTokenInfo.ChainCode
	subReq.BaseAddress = subTokenInfo.BaseAddress
	subReq.Symbol = subTokenInfo.Symbol
	subReq.NoticeType = int(subTokenInfo.NoticeType)
	subReq.TargetPrice = subTokenInfo.TargetPrice
	subReq.Data = subTokenInfo.Data
	currentPrice := util.FormatNumber(subTokenInfo.StartPrice)
	targetPrice := util.FormatNumber(subTokenInfo.TargetPrice)
	if monitorType == "price" {
		sendParams := &bot.SendMessageParams{
			ChatID:      chatId,
			Text:        fmt.Sprintf(tmplll, currentPrice, targetPrice),
			ReplyMarkup: kb,
		}
		store.BotMessageAdd()
		msg, err := b.SendMessage(ctx, sendParams)
		if err != nil {
			log.Error().Err(err).Send()
			util.QuickMessage(ctx, b, chatId, "出错了,请联系客服")
			return
		}
		subReq.SessionMessageID = msg.ID
		dataSendParams, err := json.Marshal(sendParams)
		if err != nil {
			log.Error().Err(err).Send()
			util.QuickMessage(ctx, b, chatId, "出错了,请联系客服")
			return
		}
		store.RedisSetSendMessageParams(chatId, msg.ID, dataSendParams)
	} else {
		sendParams := &bot.SendMessageParams{
			ChatID:      chatId,
			Text:        fmt.Sprintf(tmplll, subReq.Data),
			ReplyMarkup: kb,
		}
		store.BotMessageAdd()
		msg, err := b.SendMessage(ctx, sendParams)
		if err != nil {
			log.Error().Err(err).Send()
			util.QuickMessage(ctx, b, chatId, "出错了,请联系客服")
			return
		}
		subReq.SessionMessageID = msg.ID
		dataSendParams, err := json.Marshal(sendParams)
		if err != nil {
			log.Error().Err(err).Send()
			util.QuickMessage(ctx, b, chatId, "出错了,请联系客服")
			return
		}
		store.RedisSetSendMessageParams(chatId, msg.ID, dataSendParams)
	}
	newData, err := subReq.JsonB()
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}
	store.UserSetAiMonitorInfo(chatId, newData)
}

func CallbackHandlerEnableTokenAiMonitor(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatId := util.EffectId(update)
	defer func() {
		store.RedisDeleteState(chatId, "edit_current")
		store.UserDeleteAiMonitorInfo(chatId)
	}()
	dataB, has := store.UserGetAiMonitorInfo(chatId)
	if !has {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s, 启动监控", util.AdminUrl))
		log.Error().Msg("cant get user aimonitorinfo")
		return
	}

	var reqDataRaw model.AISubscribeReqData
	err := json.Unmarshal(dataB, &reqDataRaw)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s, 启动监控", util.AdminUrl))
		log.Error().Err(err).Send()
		return
	}

	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s, 启动监控", util.AdminUrl))
		log.Error().Err(err).Send()
		return
	}

	reqDataRaw.NoticeType = 1 // set to once

	_, err = api.UpdateCommonSubscribe(reqDataRaw, userInfo)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s, 启动监控", util.AdminUrl))
		log.Error().Err(err).Send()
		return
	}

	util.QuickMessageWithButton(ctx, b, chatId, "启动成功", util.BackToMainMenu())
	b.DeleteMessages(ctx, &bot.DeleteMessagesParams{
		ChatID:     chatId,
		MessageIDs: []int{reqDataRaw.SessionMessageID},
	})
}
