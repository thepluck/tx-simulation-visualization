import { useEffect, useId, useRef, useState } from "react";

type ProjectHistoryDropdownProps = {
  projects: string[];
  onSelect: (path: string) => void;
};

export default function ProjectHistoryDropdown(props: ProjectHistoryDropdownProps) {
  const { projects, onSelect } = props;
  const [open, setOpen] = useState(false);
  const menuId = useId();
  const rootRef = useRef<HTMLSpanElement | null>(null);

  useEffect(() => {
    if (!open) {
      return;
    }

    const closeOnOutsidePointer = (event: PointerEvent) => {
      if (event.target instanceof Node && rootRef.current?.contains(event.target)) {
        return;
      }
      setOpen(false);
    };
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setOpen(false);
      }
    };

    document.addEventListener("pointerdown", closeOnOutsidePointer);
    document.addEventListener("keydown", closeOnEscape);
    return () => {
      document.removeEventListener("pointerdown", closeOnOutsidePointer);
      document.removeEventListener("keydown", closeOnEscape);
    };
  }, [open]);

  const hasProjects = projects.length > 0;

  return (
    <span className="project-history-dropdown" ref={rootRef}>
      <button
        aria-controls={menuId}
        aria-expanded={open}
        aria-haspopup="menu"
        aria-label="Past projects"
        className="history-button"
        disabled={!hasProjects}
        title={hasProjects ? "Past projects" : "No past projects"}
        type="button"
        onClick={() => setOpen((current) => !current)}
      >
        Past
      </button>
      {open && (
        <span className="project-history-menu" id={menuId} role="menu">
          {projects.map((path) => (
            <button
              className="project-history-item"
              key={path}
              role="menuitem"
              title={path}
              type="button"
              onClick={() => {
                onSelect(path);
                setOpen(false);
              }}
            >
              {path}
            </button>
          ))}
        </span>
      )}
    </span>
  );
}
