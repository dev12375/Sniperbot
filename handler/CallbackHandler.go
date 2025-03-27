package handler

import (
	"context"
	"os"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/api"
	"github.com/hellodex/tradingbot/entity"
	"github.com/hellodex/tradingbot/handler/callback"
	"github.com/hellodex/tradingbot/handler/commands"
	"github.com/hellodex/tradingbot/model"
	"github.com/hellodex/tradingbot/session"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/template"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
)

func DebugMiddlewares(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, bot *bot.Bot, update *models.Update) {
		log.Debug().
			Interface("new update", update).
			Send()
		next(ctx, bot, update)
	}
}

func SetBotHandler(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, bot *bot.Bot, update *models.Update) {
		userId := util.EffectId(update)
		userInfo, err := api.GetUserProfile(userId)
		if err == nil {
			err = store.UserSetBot(userId, userInfo.Data.UUID)
			if err != nil {
				log.Error().Err(err).Send()
			}
		} else {
			log.Error().Err(err).Send()
		}

		next(ctx, bot, update)
	}
}

var limitCount = int64(75)

func Limit(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, bot *bot.Bot, update *models.Update) {
		userId := util.EffectId(update)
		_ = userId
		count, err := store.BotMessageCount()
		if err != nil {
			return
		}

		if count >= limitCount {
			callback.Alert(ctx, bot, update)
		}
		time.Sleep(1 * time.Second)

		next(ctx, bot, update)
	}
}

func GetCallbackHandler() []bot.Option {
	botOptions := []bot.Option{
		bot.WithMiddlewares(Limit),
		bot.WithMiddlewares(SetBotHandler),
		// // debug update msg
		// bot.WithMiddlewares(DebugMiddlewares),
		// start=
		bot.WithMiddlewares(commands.StartWrapMiddlewares),

		// callbackQueryMiddlewares
		bot.WithMiddlewares(WrapHandlerCallback),

		// start reflash
		bot.WithCallbackQueryDataHandler(entity.RefalshStartBalacne, bot.MatchTypeExact, commands.StartReflashInfo),

		// wallet handler
		bot.WithCallbackQueryDataHandler(entity.WALLET, bot.MatchTypeExact, callback.WalletMenuHandler),
		// bot.WithCallbackQueryDataHandler(entity.SWITCH_DEFAULT_WALLET, bot.MatchTypeExact, callback.CallbackSelectChainHandler),
		bot.WithCallbackQueryDataHandler(entity.SWITCH_DEFAULT_WALLET, bot.MatchTypeExact, callback.CallbackSelectDefaultChainWallet),
		bot.WithCallbackQueryDataHandler("swchain", bot.MatchTypePrefix, callback.CallbackListSelectWalletHandler),
		bot.WithCallbackQueryDataHandler(entity.SWITCH_PUBLIC_CHAIN, bot.MatchTypeExact, callback.CallbackSelectChainHandler),
		bot.WithCallbackQueryDataHandler("swW", bot.MatchTypePrefix, callback.CallbackConfirmSelectWallet),

		// buy and sell handler
		bot.WithCallbackQueryDataHandler("buy_", bot.MatchTypePrefix, BuyCallBackHandler),
		bot.WithCallbackQueryDataHandler("sell_", bot.MatchTypePrefix, SellCallBackHandler),
		bot.WithCallbackQueryDataHandler("selectForTrade", bot.MatchTypePrefix, MessageComfromSellectWallet),

		// transferTo handler
		bot.WithCallbackQueryDataHandler("tx_", bot.MatchTypePrefix, TransferToCallBack),

		// limit order handler
		bot.WithCallbackQueryDataHandler("order_", bot.MatchTypePrefix, CallbackLimitOrder),
		bot.WithCallbackQueryDataHandler("limitOrder_", bot.MatchTypePrefix, ConfirmLimitOrder),

		// setting handler
		bot.WithCallbackQueryDataHandler(entity.SETTING, bot.MatchTypeExact, callback.SettingHandler),
		bot.WithCallbackQueryDataHandler(entity.SETTING_SLIPPY, bot.MatchTypeExact, callback.SlippyHandler),

		// setting Assets
		bot.WithCallbackQueryDataHandler(entity.ASSETS, bot.MatchTypeExact, callback.AssetsHandler),
		bot.WithCallbackQueryDataHandler("AS_", bot.MatchTypePrefix, callback.AssetsSelectHandler),

		// setting History
		bot.WithCallbackQueryDataHandler(entity.HistoryTransfer, bot.MatchTypeExact, callback.HistoryList),
		bot.WithCallbackQueryDataHandler("list_orders", bot.MatchTypeExact, commands.OpenOrdersHandler),
		bot.WithCallbackQueryDataHandler("list_orders_history", bot.MatchTypeExact, commands.OpenOrdersHistoryHandler),
		bot.WithCallbackQueryDataHandler("list_swap", bot.MatchTypeExact, commands.TradeHistoryHandler),
		bot.WithCallbackQueryDataHandler("list_transfer", bot.MatchTypeExact, commands.TransferHistoryHandler),
		bot.WithCallbackQueryDataHandler(entity.HistoryOrder, bot.MatchTypeExact, callback.OrdersList),

		// order flow
		bot.WithCallbackQueryDataHandler(entity.OrderTrade, bot.MatchTypeExact, callback.CallbackOrderFollow),

		bot.WithCallbackQueryDataHandler(entity.BUY_SELL, bot.MatchTypeExact, callback.CallbackOrderFollow),

		// invite handler
		bot.WithCallbackQueryDataHandler(entity.InviteButton, bot.MatchTypeExact, callback.InviteHandler),
		// TODO: reflash & withdrawal handler

		bot.WithCallbackQueryDataHandler(session.UserSelectChainCache, bot.MatchTypePrefix, CallBackUserSelectChainForToken),

		// reflash
		bot.WithCallbackQueryDataHandler(entity.ReflashTokenInfo, bot.MatchTypeExact, callback.ReflashTokenInfo),

		bot.WithCallbackQueryDataHandler("go_menu", bot.MatchTypeExact, commands.MenuHandler),

		// ai monitor handler
		bot.WithCallbackQueryDataHandler(entity.AiMonitorTokenInfoSetting, bot.MatchTypeExact, callback.CallbackAiMonitorTokenInfoSetting),
		bot.WithCallbackQueryDataHandler("ai_monitor_list", bot.MatchTypeExact, callback.ListAimonitorWithButton),
		bot.WithCallbackQueryDataHandler("ai_add_monitor", bot.MatchTypeExact, callback.CallabckAiMonitorSelectType),
		bot.WithCallbackQueryDataHandler("monitor_select::", bot.MatchTypePrefix, callback.CallbackAimonitorConfirmType),
		bot.WithCallbackQueryDataHandler("aiSchain", bot.MatchTypePrefix, callback.CallbackUserConfirmChainForAimonitor),
		bot.WithCallbackQueryDataHandler("add_token_alert", bot.MatchTypeExact, callback.Callback_add_token_alert),
		bot.WithCallbackQueryDataHandler("cancel_settings", bot.MatchTypeExact, callback.Callback_cancel_settings),
		bot.WithCallbackQueryDataHandler("toggle_", bot.MatchTypePrefix, callback.CallbackToggleChannels),
		bot.WithCallbackQueryDataHandler("freq_", bot.MatchTypePrefix, callback.CallbackUpdateFreqSetting),
		bot.WithCallbackQueryDataHandler("pusher_", bot.MatchTypePrefix, callback.CallbackHandleAimonitorPuserButton),
		bot.WithCallbackQueryDataHandler("ai_sendNewupdate", bot.MatchTypeExact, callback.CallbackPusherQuickTradeButton),
		bot.WithCallbackQueryDataHandler(entity.AIMonitorButton, bot.MatchTypeExact, callback.CallbackAIMonitorMenu),

		bot.WithCallbackQueryDataHandler(entity.Withdrawal, bot.MatchTypeExact, callback.CallbackWithdrawl),
		bot.WithCallbackQueryDataHandler("withdrawal_", bot.MatchTypePrefix, callback.CallBackWithdrawlButton),

		bot.WithCallbackQueryDataHandler("view_order::", bot.MatchTypePrefix, template.CallbackHandlerViewOrder),
		bot.WithCallbackQueryDataHandler("backToOrderList", bot.MatchTypeExact, commands.OpenOrdersHandler),
		bot.WithCallbackQueryDataHandler("backToMainMenu", bot.MatchTypeExact, commands.StartHandler),
		bot.WithCallbackQueryDataHandler("cancelOrder::", bot.MatchTypePrefix, template.CallabckOrderCancelOrder),

		bot.WithCallbackQueryDataHandler("edit_current_", bot.MatchTypePrefix, callback.CallBackEditCurrentAimonitor),
		bot.WithCallbackQueryDataHandler("save_current_", bot.MatchTypePrefix, callback.CallbackSaveCurrentAimonitor),
		bot.WithCallbackQueryDataHandler("pause_current_", bot.MatchTypePrefix, callback.CallBackPauseCurrentAimonitor),
		bot.WithCallbackQueryDataHandler("enable_current_", bot.MatchTypePrefix, callback.CallbackHandlerEnableTokenAiMonitor),
		bot.WithCallbackQueryDataHandler("delete_current_", bot.MatchTypePrefix, callback.CallBackDeleteCurrentAimonitor),
		bot.WithCallbackQueryDataHandler("aiL:", bot.MatchTypePrefix, callback.CallbackHendlerSelectEditAimonitor),
	}

	return botOptions
}

func WrapHandlerCallback(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID := util.EffectId(update)

		// handle user send new token address
		if update.Message != nil && update.Message.Text != "" {
			if util.IsCryproAddress(update.Message.Text) {
				session.GetSessionManager().Set(chatID, session.UserSelectTokenAddressCache, update.Message.Text)
				v, has := store.Get(chatID, store.TradeSession)
				if has {
					wrapMessage, ok := v.(*model.MessageWrap)
					if ok {
						log.Debug().Msg("user send new token")
						if _, err := b.UnpinAllChatMessages(ctx, &bot.UnpinAllChatMessagesParams{
							ChatID: chatID,
						}); err == nil {
							b.DeleteMessage(ctx, &bot.DeleteMessageParams{
								ChatID:    chatID,
								MessageID: wrapMessage.Message.ID,
							})
						}
						next(ctx, b, update)
						return
					}
				}
			}

			// handle user callback old button
		} else if update.CallbackQuery != nil {
			callback := update.CallbackQuery
			util.CallBackAnswer(ctx, b, callback)

			onCallbackTime := callback.Message.Message.Date
			initTime := os.Getenv("BOT_INIT_TIMESTAMPS")
			if initTime != "" {
				initTimeT := time.Unix(cast.ToInt64(initTime), 0)

				targetTime := time.Unix(int64(onCallbackTime), 0)
				if targetTime.Before(initTimeT) {
					log.Debug().Msg("old message ignored")
					commands.StartHandler(ctx, b, update)
					return
				}
			}

			v, has := store.Get(chatID, store.TradeSession)
			if has {
				wrapMessage, ok := v.(*model.MessageWrap)
				log.Debug().Interface("wrapMessage", wrapMessage)
				if ok {
					log.Debug().Int("stored_message_id", wrapMessage.Message.ID).
						Int("callback_message_id", callback.Message.Message.ID).
						Msg("Comparing message IDs")
					if wrapMessage.Message.ID == callback.Message.Message.ID {
						if wrapMessage.IsExpired() {
							log.Debug().Msg("CallbackQuery button expired")
							if _, err := b.UnpinAllChatMessages(ctx, &bot.UnpinAllChatMessagesParams{
								ChatID: chatID,
							}); err != nil {
								log.Error().Err(err).Send()
							}
							b.DeleteMessage(ctx, &bot.DeleteMessageParams{
								ChatID:    chatID,
								MessageID: wrapMessage.Message.ID,
							})
							commands.StartHandler(ctx, b, update)
							return
						}
					}
				}
			}
		}

		next(ctx, b, update)
	}
}
