// Ghost FTP by Brendigo
import type { ComponentProps } from "react";
import { ConfirmModal as FileConfirmModal } from "@ghostftp/file-ui";
import { toastError } from "@/lib/errors";

type Props = Omit<ComponentProps<typeof FileConfirmModal>, "onError">;

export function ConfirmModal({ confirmLabel = "Confirm", ...props }: Props) {
  return (
    <FileConfirmModal
      {...props}
      confirmLabel={confirmLabel}
      onError={(error) => toastError(error, `Couldn't ${confirmLabel.toLowerCase()}`)}
    />
  );
}
