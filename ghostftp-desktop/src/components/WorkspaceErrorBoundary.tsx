import React from "react";
import { AlertTriangle, FolderOpen, RefreshCw } from "lucide-react";

interface Props extends React.PropsWithChildren {
  label: string;
  resetKey: string;
  onReturnToFiles: () => void;
}

interface State {
  failed: boolean;
}

function safeWorkspaceDiagnostic(error: unknown): string {
  const raw = error instanceof Error ? error.message : String(error);
  return raw
    .replace(/[A-Za-z]:\\Users\\[^\\\s]+/gi, "C:\\Users\\<user>")
    .replace(/\/home\/[^/\s]+/g, "/home/<user>")
    .replace(/([?&](?:token|code|password|passphrase|secret|key)=)[^&\s]+/gi, "$1<redacted>")
    .replace(/([a-z][a-z0-9+.-]*:\/\/[^:\s/@]+:)[^@\s/]+@/gi, "$1<redacted>@")
    .slice(0, 320);
}

/**
 * Contains synchronous render failures to the active workspace so the
 * persistent title bar, sidebar and application shell stay usable.
 */
export class WorkspaceErrorBoundary extends React.Component<Props, State> {
  state: State = { failed: false };

  static getDerivedStateFromError(): State {
    return { failed: true };
  }

  componentDidCatch(error: unknown) {
    console.error(
      `Ghost FTP ${this.props.label} workspace failure:`,
      safeWorkspaceDiagnostic(error)
    );
  }

  componentDidUpdate(prevProps: Props) {
    if (prevProps.resetKey !== this.props.resetKey && this.state.failed) {
      this.setState({ failed: false });
    }
  }

  private retry = () => {
    this.setState({ failed: false });
  };

  private returnToFiles = () => {
    this.setState({ failed: false }, this.props.onReturnToFiles);
  };

  render() {
    if (!this.state.failed) return this.props.children;

    const inFiles = this.props.label === "Files";

    return (
      <section
        className="flex h-full min-h-0 items-center justify-center bg-[#041425] p-6 text-[#eaf6ff]"
        role="alert"
        aria-live="assertive"
      >
        <div className="w-full max-w-xl rounded-xl border border-[#245b84] bg-[#071d32] p-6 shadow-2xl">
          <div className="flex items-start gap-3">
            <div className="rounded-lg border border-[#2f6388] bg-[#0a2944] p-2 text-[#7dccff]">
              <AlertTriangle size={20} />
            </div>
            <div className="min-w-0">
              <h2 className="text-lg font-semibold">{this.props.label} couldn't be displayed</h2>
              <p className="mt-2 text-sm leading-6 text-[#9fc3db]">
                The interface error was contained to this workspace. Retry the view
                {inFiles ? "." : " or return to Files; the main Ghost FTP window stays open."}
              </p>
            </div>
          </div>
          <div className="mt-5 flex flex-wrap gap-2">
            <button type="button" className="ghost-primary-button" onClick={this.retry}>
              <RefreshCw size={14} /> Try again
            </button>
            {!inFiles && (
              <button type="button" className="ghost-mini-button" onClick={this.returnToFiles}>
                <FolderOpen size={14} /> Return to Files
              </button>
            )}
          </div>
        </div>
      </section>
    );
  }
}
