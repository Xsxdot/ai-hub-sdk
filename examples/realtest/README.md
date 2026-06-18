# ai-hub SDK realtest

This example runs real smoke calls against an ai-hub HTTP service.

It intentionally reads the API key from the environment so keys are not committed.

## Run

```bash
AIHUB_API_KEY="your-key" \
AIHUB_BASE_URL="http://localhost:10100" \
AIHUB_RUN_STREAM=true \
go run ./examples/realtest
```

Defaults match the local test logical models:

- `AIHUB_CHAT_MODEL=normal-chat`
- `AIHUB_IMAGE_MODEL=image`
- `AIHUB_VIDEO_MODEL=video`
- `AIHUB_ASR_MODEL=asr`

The command always attempts chat, image, video submit/get, and ASR. With `AIHUB_RUN_STREAM=true`, it also exercises the SSE chat route. It continues after a failed step so every modality reports its own result.

Optional stream and video waiting:

```bash
AIHUB_API_KEY="your-key" \
AIHUB_RUN_STREAM=true \
AIHUB_WAIT_VIDEO=true \
AIHUB_VIDEO_TIMEOUT=10m \
go run ./examples/realtest
```

Useful overrides:

- `AIHUB_AUDIO_URL`: ASR input URL. Defaults to a small public JFK sample audio, but provider-side fetches can time out against GitHub raw URLs. For stable real ASR smoke, use an OSS or otherwise provider-reachable URL.
- `AIHUB_CHAT_PROMPT`
- `AIHUB_IMAGE_PROMPT`
- `AIHUB_VIDEO_PROMPT`
- `AIHUB_HTTP_TIMEOUT`, such as `2m`

## Local smoke notes

Observed against `http://localhost:10100` on 2026-06-18 with the logical models above:

- Chat passed.
- Chat stream passed with `AIHUB_RUN_STREAM=true`.
- Image passed and returned a `media/image/.../0.png` artifact reference.
- Video reached `/v1/videos/jobs`, then the server returned `500` because the upstream distributor had no configured channel for the mapped `veo_3_1` model.
- ASR reached `/v1/audio/transcriptions`, then the upstream `qwen3-asr-flash` call failed because it timed out downloading the default GitHub sample audio. Re-run with `AIHUB_AUDIO_URL` pointing at a provider-reachable audio file.
