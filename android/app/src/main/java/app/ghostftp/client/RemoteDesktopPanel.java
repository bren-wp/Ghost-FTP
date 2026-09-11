package app.ghostftp.client;

import android.app.Activity;
import android.content.Intent;
import android.graphics.Color;
import android.net.Uri;
import android.view.ViewGroup;
import android.widget.Button;
import android.widget.EditText;
import android.widget.LinearLayout;
import android.widget.TextView;

final class RemoteDesktopPanel {
    private RemoteDesktopPanel() { }

    static LinearLayout create(Activity activity, EditText connectionHost) {
        LinearLayout panel = new LinearLayout(activity);
        panel.setOrientation(LinearLayout.VERTICAL);
        panel.setPadding(0, dp(activity, 18), 0, dp(activity, 8));

        TextView title = new TextView(activity);
        title.setText("ADVANCED · REMOTE DESKTOP");
        title.setTextSize(15);
        title.setTextColor(Color.rgb(94, 214, 200));
        panel.addView(title);

        TextView help = new TextView(activity);
        help.setText("Open your own RDP server in an installed Remote Desktop app. Ghost FTP never stores or forwards the RDP password.");
        help.setTextSize(13);
        help.setTextColor(Color.LTGRAY);
        panel.addView(help);

        EditText target = new EditText(activity);
        target.setHint("RDP server (host[:port])");
        target.setSingleLine(true);
        target.setTextColor(Color.WHITE);
        target.setHintTextColor(Color.GRAY);
        panel.addView(target, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));

        Button open = new Button(activity);
        open.setText("Open Remote Desktop");
        open.setAllCaps(false);
        panel.addView(open, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));

        TextView status = new TextView(activity);
        status.setText("Authentication stays inside your RDP client.");
        status.setTextSize(12);
        status.setTextColor(Color.LTGRAY);
        panel.addView(status);

        open.setOnClickListener(v -> {
            String value = target.getText().toString().trim();
            if (value.isEmpty() && connectionHost != null) value = connectionHost.getText().toString().trim();
            final Uri uri;
            try {
                uri = RemoteDesktopTarget.uri(value);
            } catch (IllegalArgumentException e) {
                status.setText(e.getMessage());
                return;
            }
            Intent intent = new Intent(Intent.ACTION_VIEW, uri);
            if (intent.resolveActivity(activity.getPackageManager()) == null) {
                status.setText("No RDP client is installed. Install a Remote Desktop app that registers the rdp:// scheme.");
                return;
            }
            try {
                activity.startActivity(intent);
                status.setText("Remote Desktop opened. Authentication remains inside the RDP client.");
            } catch (RuntimeException e) {
                status.setText("Remote Desktop could not be opened.");
            }
        });
        return panel;
    }

    private static int dp(Activity activity, int value) {
        return Math.round(value * activity.getResources().getDisplayMetrics().density);
    }
}
