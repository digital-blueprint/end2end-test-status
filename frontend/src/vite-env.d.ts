/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_PATH_PREFIX?: string
  readonly VITE_FOOTER_TEXT?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
