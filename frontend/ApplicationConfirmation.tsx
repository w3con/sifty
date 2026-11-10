import React, { useEffect, useRef, useState } from "react";
import type { Job } from "./model/core";
import type { OutcomeReason } from "./model/workflow";

export function ApplicationConfirmation({
  job,
  onClose,
  onConfirm,
  onDismiss,
}: {
  job: Job;
  onClose: () => void;
  onConfirm: () => Promise<boolean>;
  onDismiss: (reason: OutcomeReason) => Promise<boolean>;
}) {
  const dialog = useRef<HTMLDialogElement>(null),
    [pending, setPending] = useState<string | null>(null),
    [failed, setFailed] = useState(false);
  const saving = pending !== null;
  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null,
      element = dialog.current;
    element?.showModal();
    return () => {
      element?.close();
      if (previous?.isConnected) previous.focus();
    };
  }, []);
  async function save(kind: string, action: () => Promise<boolean>) {
    if (saving) return;
    setPending(kind);
    setFailed(false);
    try {
      if (await action()) onClose();
      else setFailed(true);
    } catch {
      setFailed(true);
    } finally {
      setPending(null);
    }
  }
  return (
    <dialog
      ref={dialog}
      className="application-confirmation"
      aria-labelledby="application-question"
      aria-describedby="application-explanation"
      onCancel={(e) => {
        // Escape is a native dialog cancellation; this modal is dismissible only
        // by an explicit button action.
        e.preventDefault();
      }}
    >
      <h2 id="application-question">Did you submit your application?</h2>
      <p>
        {job.title} · {job.company}
      </p>
      <p id="application-explanation" className="muted">
        Opening the employer page does not submit an application. Confirm only
        after you have submitted it yourself. You can also confirm later from
        Pre-selected. Choose a negative outcome only if you will not apply.
      </p>
      {failed && (
        <p className="error" role="alert">
          The outcome could not be saved. The role has not been changed by this
          action. Try again, or close this dialog to review the error.
        </p>
      )}
      <div className="confirmation-actions">
        <button type="button" disabled={saving} onClick={onClose}>
          Not yet
        </button>
        <button
          type="button"
          disabled={saving}
          onClick={() => void save("stale", () => onDismiss("stale"))}
        >
          {pending === "stale" ? "Saving…" : "No, vacancy is stale"}
        </button>
        <button
          type="button"
          disabled={saving}
          onClick={() => void save("closed", () => onDismiss("closed"))}
        >
          {pending === "closed" ? "Saving…" : "No, vacancy is closed"}
        </button>
        <button
          type="button"
          disabled={saving}
          onClick={() => void save("not_suitable", () => onDismiss("not_suitable"))}
        >
          {pending === "not_suitable" ? "Saving…" : "No, not suitable"}
        </button>
        <button
          type="button"
          className="primary"
          disabled={saving}
          onClick={() => void save("applied", onConfirm)}
        >
          {pending === "applied" ? "Saving…" : "Yes, applied"}
        </button>
      </div>
    </dialog>
  );
}
