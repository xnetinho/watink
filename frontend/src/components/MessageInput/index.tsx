/* @jsxImportSource react */
import React, { useEffect, useRef } from "react";
import clsx from "clsx";
import { Send, Mic, XOctagon, CheckCircle } from "lucide-react";
import { Loader2 } from "lucide-react";

import RecordingTimer from "./RecordingTimer";
import { i18n } from "../../translate/i18n";
import { useThemeContext } from "../../context/DarkMode";
import { useMediaDrop } from "../../context/MediaDrop/MediaDropContext";

import { useMessageInput } from "./hooks/useMessageInput";
import ReplyPreview from "./components/ReplyPreview";
import MediaPreviewPanel from "./components/MediaPreviewPanel";
import InputToolbar from "./components/InputToolbar";
import QuickAnswersList from "./components/QuickAnswersList";
import MentionsList from "./components/MentionsList";


const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
export interface ReplyPolicyUI {
  canReply?: boolean;
  reason?: string;
  message?: string;
  validUntil?: string;
  capabilities?: {
    maxTextBytes?: number;
    groups?: boolean;
    media?: string[];
  };
}

interface MessageInputProps {
  ticketStatus: string;
  whatsappStatus?: string;
  replyPolicy?: ReplyPolicyUI;
}

const MessageInput: React.FC<MessageInputProps> = ({ ticketStatus, whatsappStatus, replyPolicy }) => {
  const { appTheme } = useThemeContext();

  const {
    medias,
    inputMessage,
    showEmoji,
    loading,
    recording,
    quickAnswers,
    typeBar,
    anchorElMenu,
    mentions,
    mentionOpen,
    signMessage,
    replyingMessage,
    inputRef,
    setShowEmoji,
    setAnchorElMenu,
    setReplyingMessage,
    setSignMessage,
    handleChangeInput,
    handleQuickAnswersClick,
    handleAddEmoji,
    handleChangeMedias,
    handleInputPaste,
    handleDropFiles,
    handleMediaCaptionChange,
    handleAudioKindChange,
    handleUploadMedia,
    handleRemoveMedia,
    handleClearMedias,
    handleSendMessage,
    handleStartRecording,
    handleUploadAudio,
    handleCancelAudio,
    handleMentionClick,
  } = useMessageInput();

  // A área da conversa (que cobre a lista inteira) entrega aqui os arquivos soltos. O ref mantém a função sempre atual,
  // sem refazer o registro a cada digitação (o texto digitado vira a legenda do primeiro arquivo).
  const { register } = useMediaDrop();
  const dropRef = useRef(handleDropFiles);
  dropRef.current = handleDropFiles;
  useEffect(() => register((files) => dropRef.current(files)), [register]);

  if (ticketStatus !== "open") {
    return (
      <div className="flex h-[62px] w-full shrink-0 items-center justify-center border-t border-[var(--border-divider)] bg-[hsl(var(--bg-surface-alt))] px-4">
        <span className="text-sm text-muted-foreground">
          {i18n.t("messagesInput.placeholderClosed")}
        </span>
      </div>
    );
  }

  if (replyPolicy && replyPolicy.canReply === false) {
    return (
      <div className="w-full shrink-0 border-t border-[var(--border-divider)] bg-[hsl(var(--bg-surface-alt))] px-4 py-3">
        <div className="w-full text-center bg-amber-500/10 text-amber-700 dark:text-amber-400 border border-amber-500/20 rounded-lg p-3 flex flex-col items-center">
          <span className="font-bold text-xs uppercase tracking-wider">Resposta Bloqueada pelo Canal</span>
          <span className="text-xs mt-1">
            {replyPolicy.message || tr("uiText.messageInput.replyWindowClosed")}
          </span>
        </div>
      </div>
    );
  }

  if (whatsappStatus && whatsappStatus !== "CONNECTED") {
    return (
      <div className="w-full shrink-0 border-t border-[var(--border-divider)] bg-[hsl(var(--bg-surface-alt))] px-4 py-3">
        <div className="w-full text-center bg-[hsl(var(--status-error-bg))] text-[hsl(var(--status-error-text))] rounded-lg p-3 flex flex-col items-center">
          <span className="font-bold text-sm">{tr("uiText.messageInput.connectionDown")}</span>
          <span className="text-xs mt-1">
            {tr("uiText.messageInput.connectionDownHint")}
          </span>
        </div>
      </div>
    );
  }

  if (medias.length > 0) {
    return (
      <MediaPreviewPanel
        medias={medias}
        loading={loading}
        onRemoveMedia={handleRemoveMedia}
        onCaptionChange={handleMediaCaptionChange}
        onAudioKindChange={handleAudioKindChange}
        onAddMore={handleChangeMedias}
        onCancelAll={handleClearMedias}
        onUpload={handleUploadMedia}
      />
    );
  }

  return (
    <div className="w-full shrink-0 flex flex-col border-t border-[var(--border-divider)] bg-[hsl(var(--bg-surface-alt))]">
      {replyingMessage && (
        <ReplyPreview
          message={replyingMessage}
          loading={loading}
          ticketStatus={ticketStatus}
          onCancel={() => setReplyingMessage(null)}
        />
      )}

      <div className="bg-[hsl(var(--bg-surface-alt))] w-full flex px-3 py-2.5 items-center gap-1">
        <InputToolbar
          loading={loading}
          recording={recording}
          ticketStatus={ticketStatus}
          showEmoji={showEmoji}
          signMessage={Boolean(signMessage)}
          anchorElMenu={anchorElMenu}
          onToggleEmoji={() => setShowEmoji(!showEmoji)}
          onEmojiClose={() => setShowEmoji(false)}
          onEmojiSelect={handleAddEmoji}
          onMediaChange={handleChangeMedias}
          onSignChange={setSignMessage}
          onMenuOpenChange={setAnchorElMenu}
        />

        {/* Input area */}
        <div
          className={clsx(
            "px-4 py-2.5 bg-[hsl(var(--bg-surface))] border border-[var(--border-subtle)] flex items-center rounded-full flex-1 relative min-h-[48px]",
            appTheme === "saas" && "shadow-none"
          )}
        >
          <textarea
            ref={inputRef}
            className="flex-1 border-none bg-transparent resize-none outline-none text-[15px] leading-[1.5] min-h-[24px] max-h-[160px]"
            placeholder={
              ticketStatus === "open"
                ? i18n.t("messagesInput.placeholderOpen") as string
                : i18n.t("messagesInput.placeholderClosed") as string
            }
            rows={1}
            value={inputMessage}
            onChange={handleChangeInput}
            disabled={recording || loading || ticketStatus !== "open"}
            onPaste={(e) => ticketStatus === "open" && handleInputPaste(e)}
            onKeyPress={(e) => {
              if (loading || e.shiftKey) return;
              if (e.key === "Enter") {
                e.preventDefault();
                handleSendMessage();
              }
            }}
          />

          {typeBar && (
            <QuickAnswersList
              answers={quickAnswers}
              onSelect={handleQuickAnswersClick}
            />
          )}

          {mentionOpen && (
            <MentionsList
              mentions={mentions}
              onSelect={handleMentionClick}
            />
          )}
        </div>

        {/* Action buttons */}
        {inputMessage ? (
          <button
            type="button"
            className="p-2.5 rounded-full hover:bg-[var(--border-subtle)] transition-colors disabled:opacity-50"
            onClick={handleSendMessage}
            disabled={loading}
            aria-label="Enviar mensagem"
          >
            <Send className="h-6 w-6 text-gray-500" />
          </button>
        ) : recording ? (
          <div className="flex items-center">
            <button
              type="button"
              className="p-2 rounded-full hover:bg-[var(--border-subtle)] transition-colors disabled:opacity-50"
              disabled={loading}
              onClick={handleCancelAudio}
              aria-label={tr("uiText.messageInput.cancelRecording")}
            >
              <XOctagon className="h-5 w-5 text-red-500" />
            </button>
            {loading ? (
              <Loader2 className="h-6 w-6 animate-spin text-[hsl(var(--status-success))] opacity-70" />
            ) : (
              <RecordingTimer />
            )}
            <button
              type="button"
              className="p-2 rounded-full hover:bg-[var(--border-subtle)] transition-colors disabled:opacity-50"
              onClick={handleUploadAudio}
              disabled={loading}
              aria-label={tr("uiText.messageInput.sendRecording")}
            >
              <CheckCircle className="h-5 w-5 text-green-500" />
            </button>
          </div>
        ) : (
          <button
            type="button"
            className="p-2 rounded-full hover:bg-[var(--border-subtle)] transition-colors disabled:opacity-50"
            disabled={loading || ticketStatus !== "open"}
            onClick={handleStartRecording}
            aria-label={tr("uiText.messageInput.record")}
          >
            <Mic className="h-6 w-6 text-gray-500" />
          </button>
        )}
      </div>
    </div>
  );
};

export default MessageInput;
