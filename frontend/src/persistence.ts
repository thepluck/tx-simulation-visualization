import { defaults, type FormState, type RequestTab } from "./form";

const storageKey = "txsim.ui.v1";

export type PersistedUIState = {
  form: FormState;
  requestTab: RequestTab;
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
    return {
      form: { ...defaults, ...(parsed.form ?? {}) },
      requestTab: validRequestTab(parsed.requestTab) ? parsed.requestTab : "overrides",
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
  window.localStorage.setItem(storageKey, JSON.stringify(state));
}

function defaultUIState(): PersistedUIState {
  return {
    form: defaults,
    requestTab: "overrides",
    traceExpandDepth: 3
  };
}

function validRequestTab(value: unknown): value is RequestTab {
  return value === "overrides" || value === "state" || value === "compiler";
}

function clampDepth(value: unknown): number {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) {
    return 3;
  }
  return Math.min(20, Math.max(0, Math.trunc(parsed)));
}
