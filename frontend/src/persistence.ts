import { defaults, type FormState, type OutputView, type RequestTab } from "./form";
import { simulateResponseSchema } from "./schemas";
import type { SimulateResponse } from "./types";

const storageKey = "txsim.ui.v1";

export type PersistedUIState = {
  form: FormState;
  outputView: OutputView;
  requestTab: RequestTab;
  response: SimulateResponse | null;
  traceExpandDepth: number;
};

export function loadPersistedUIState(): PersistedUIState {
  if (typeof window === "undefined") {
    return defaultUIState();
  }

  const raw = window.localStorage.getItem(storageKey);
  if (!raw) {
    return defaultUIState();
  }

  try {
    const parsed = JSON.parse(raw) as Partial<PersistedUIState>;
    const response = sanitizeResponse(parsed.response);
    const outputView = validOutputView(parsed.outputView) && response ? parsed.outputView : "trace";
    return {
      form: sanitizeForm(parsed.form),
      outputView,
      requestTab: validRequestTab(parsed.requestTab) ? parsed.requestTab : "overrides",
      response,
      traceExpandDepth: clampDepth(parsed.traceExpandDepth)
    };
  } catch {
    return defaultUIState();
  }
}

export function savePersistedUIState(state: PersistedUIState) {
  if (typeof window === "undefined") {
    return;
  }
  try {
    window.localStorage.setItem(storageKey, JSON.stringify(state));
  } catch {
    try {
      window.localStorage.setItem(storageKey, JSON.stringify({ ...state, response: null }));
    } catch {
      // Persisting is only a debugging convenience; never block the UI on quota issues.
    }
  }
}

function sanitizeForm(value: unknown): FormState {
  if (!value || typeof value !== "object") {
    return defaults;
  }

  const input = value as Partial<FormState>;
  const form = { ...defaults };
  for (const key of Object.keys(defaults) as Array<keyof FormState>) {
    const next = input[key];
    if (next !== undefined) {
      form[key] = next as never;
    }
  }
  return form;
}

function defaultUIState(): PersistedUIState {
  return {
    form: defaults,
    outputView: "trace",
    requestTab: "overrides",
    response: null,
    traceExpandDepth: 3
  };
}

function validRequestTab(value: unknown): value is RequestTab {
  return value === "overrides" || value === "state" || value === "compiler";
}

function validOutputView(value: unknown): value is OutputView {
  return value === "trace" || value === "flow" || value === "balances" || value === "json";
}

function sanitizeResponse(value: unknown): SimulateResponse | null {
  const parsed = simulateResponseSchema.safeParse(value);
  return parsed.success ? parsed.data : null;
}

function clampDepth(value: unknown): number {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) {
    return 3;
  }
  return Math.min(20, Math.max(0, Math.trunc(parsed)));
}
