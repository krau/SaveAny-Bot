# Web UI conventions

- Transient action results and validation errors MUST use `useToastStore` / `useFormFeedback`. Do not insert error or success banners into the document flow; they shift the interface.
- Field validation MUST call `fail(message, elementId)` and can use the shared `.field.is-invalid` state. Clear the state on input.
- Persistent domain state such as a task's stored error or a connection failure may remain inline.
- Async buttons MUST use `LoadingButton`; keep the normal label and let the component overlay a spinner. Never replace labels with longer "saving..." text or animate button width.
- Destructive or irreversible actions MUST use `useConfirmStore`; never call `window.confirm`.
- Motion MUST use the tokens in `src/assets/main.css`, animate only transform/opacity/filter where possible, and keep the global `prefers-reduced-motion` fallback intact.
- Theme-aware colors MUST use the CSS variables in `src/assets/main.css`; never hardcode a light-only surface or text color. Theme preference is managed by `useThemeStore`.
- Modals whose form content changes must tween between content heights instead of using a tall fixed placeholder; keep the actions pinned and scroll only when the viewport constrains the form.
- Use `pnpm` for every install and script. Do not add npm lockfiles.
