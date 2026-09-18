package app.ghostftp.client;

import android.app.Activity;
import android.content.Context;
import android.content.res.ColorStateList;
import android.graphics.Color;
import android.graphics.Typeface;
import android.graphics.drawable.GradientDrawable;
import android.graphics.drawable.RippleDrawable;
import android.view.Gravity;
import android.view.View;
import android.view.ViewGroup;
import android.widget.ArrayAdapter;
import android.widget.Button;
import android.widget.EditText;
import android.widget.ListView;
import android.widget.Spinner;
import android.widget.TextView;

import java.util.List;

final class GhostTheme {
    static int WINDOW;
    static int PANEL;
    static int LIST;
    static int BORDER;
    static int TEXT;
    static int MUTED;
    static int ACCENT;
    static int ACCENT_STRONG;
    static int ON_ACCENT;
    static int SUCCESS;
    static int WARN;
    static int DANGER;
    static int SELECTION;

    static final String APPEARANCE_DARK = "dark";
    static final String APPEARANCE_LIGHT = "light";

    private static boolean light;

    private GhostTheme() {
    }

    static String normalizeAppearance(String value) {
        return value != null && APPEARANCE_LIGHT.equalsIgnoreCase(value.trim())
                ? APPEARANCE_LIGHT
                : APPEARANCE_DARK;
    }

    static void apply(Context context) {
        apply(context, APPEARANCE_DARK);
    }

    static void apply(Context context, String appearance) {
        light = APPEARANCE_LIGHT.equals(normalizeAppearance(appearance));
        if (light) {
            WINDOW = Color.rgb(0xEE, 0xF1, 0xF5);
            PANEL = Color.rgb(0xF6, 0xF8, 0xFB);
            LIST = Color.rgb(0xFA, 0xFB, 0xFD);
            BORDER = Color.rgb(0xD6, 0xDC, 0xE5);
            TEXT = Color.rgb(0x17, 0x20, 0x33);
            MUTED = Color.rgb(0x66, 0x70, 0x85);
            ACCENT = Color.rgb(0xA6, 0x65, 0x00);
            ACCENT_STRONG = Color.rgb(0x87, 0x51, 0x00);
            ON_ACCENT = Color.rgb(0xFA, 0xFB, 0xFD);
            SUCCESS = Color.rgb(0x1B, 0x7F, 0x4B);
            WARN = Color.rgb(0x9A, 0x67, 0x00);
            DANGER = Color.rgb(0xB4, 0x23, 0x18);
            SELECTION = Color.rgb(0xF5, 0xE7, 0xC7);
            return;
        }

        WINDOW = Color.rgb(0x0B, 0x0F, 0x17);
        PANEL = Color.rgb(0x12, 0x18, 0x24);
        LIST = Color.rgb(0x16, 0x1D, 0x2A);
        BORDER = Color.rgb(0x2C, 0x36, 0x48);
        TEXT = Color.rgb(0xF2, 0xF5, 0xFA);
        MUTED = Color.rgb(0x97, 0xA3, 0xB8);
        ACCENT = Color.rgb(0xF6, 0xC4, 0x45);
        ACCENT_STRONG = Color.rgb(0xFF, 0xD7, 0x68);
        ON_ACCENT = Color.rgb(0x16, 0x13, 0x0B);
        SUCCESS = Color.rgb(0x4A, 0xD7, 0x9B);
        WARN = Color.rgb(0xF2, 0xBA, 0x55);
        DANGER = Color.rgb(0xFF, 0x68, 0x78);
        SELECTION = Color.rgb(0x2B, 0x25, 0x15);
    }

    static void applySystemBars(Activity activity) {
        activity.getWindow().setStatusBarColor(WINDOW);
        activity.getWindow().setNavigationBarColor(WINDOW);
        View decor = activity.getWindow().getDecorView();
        int flags = decor.getSystemUiVisibility();
        int lightFlags = View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR | View.SYSTEM_UI_FLAG_LIGHT_NAVIGATION_BAR;
        flags = light ? (flags | lightFlags) : (flags & ~lightFlags);
        decor.setSystemUiVisibility(flags);
    }

    static void stylePrimaryButton(Button button) {
        styleButton(button, ACCENT, ON_ACCENT, ACCENT_STRONG);
    }

    static void styleSecondaryButton(Button button) {
        styleButton(button, LIST, TEXT, SELECTION);
    }

    static void styleDangerButton(Button button) {
        int fill = light ? Color.rgb(0xFA, 0xEA, 0xE8) : Color.rgb(0x3A, 0x1D, 0x27);
        int ripple = light ? Color.rgb(0xF4, 0xD5, 0xD1) : Color.rgb(0x51, 0x25, 0x31);
        styleButton(button, fill, DANGER, ripple);
    }

    static void styleButton(Button button, int fill, int textColor, int rippleColor) {
        button.setAllCaps(false);
        button.setTextColor(textColor);
        button.setTextSize(13f);
        button.setTypeface(Typeface.DEFAULT, Typeface.BOLD);
        button.setGravity(Gravity.CENTER);
        button.setMinHeight(dp(button.getContext(), 44));
        button.setPadding(dp(button.getContext(), 10), 0, dp(button.getContext(), 10), 0);
        button.setBackground(ripple(button.getContext(), fill, BORDER, rippleColor, 10));
    }

    static void styleField(EditText edit) {
        edit.setTextColor(TEXT);
        edit.setHintTextColor(MUTED);
        edit.setTextSize(14f);
        edit.setSingleLine(true);
        edit.setMinHeight(dp(edit.getContext(), 48));
        edit.setPadding(dp(edit.getContext(), 12), dp(edit.getContext(), 4), dp(edit.getContext(), 12), dp(edit.getContext(), 4));
        edit.setBackground(rounded(edit.getContext(), LIST, BORDER, 10));
    }

    static void styleSpinner(Spinner spinner) {
        spinner.setPadding(dp(spinner.getContext(), 8), dp(spinner.getContext(), 6), dp(spinner.getContext(), 8), dp(spinner.getContext(), 6));
        spinner.setBackground(rounded(spinner.getContext(), LIST, BORDER, 10));
        spinner.setPopupBackgroundDrawable(rounded(spinner.getContext(), PANEL, BORDER, 10));
    }

    static void styleList(ListView list) {
        list.setBackground(rounded(list.getContext(), LIST, BORDER, 12));
        list.setDividerHeight(0);
        list.setPadding(dp(list.getContext(), 4), dp(list.getContext(), 4), dp(list.getContext(), 4), dp(list.getContext(), 4));
        list.setClipToPadding(false);
        list.setSelector(rounded(list.getContext(), SELECTION, ACCENT, 8));
    }

    static void styleBadge(TextView view, int color) {
        view.setTextColor(color);
        view.setTextSize(11f);
        view.setTypeface(Typeface.DEFAULT, Typeface.BOLD);
        view.setGravity(Gravity.CENTER);
        view.setPadding(dp(view.getContext(), 10), dp(view.getContext(), 6), dp(view.getContext(), 10), dp(view.getContext(), 6));
        view.setBackground(rounded(view.getContext(), SELECTION, color, 999));
    }

    static ArrayAdapter<String> spinnerAdapter(Context context, List<String> values) {
        return new GhostArrayAdapter(context, values, true);
    }

    static ArrayAdapter<String> listAdapter(Context context, List<String> values) {
        return new GhostArrayAdapter(context, values, false);
    }

    static GradientDrawable rounded(Context context, int fill, int stroke, int radiusDp) {
        GradientDrawable drawable = new GradientDrawable();
        drawable.setColor(fill);
        drawable.setCornerRadius(dp(context, radiusDp));
        drawable.setStroke(dp(context, 1), stroke);
        return drawable;
    }

    static int statusColor(String status) {
        String value = status == null ? "" : status.toLowerCase(java.util.Locale.ROOT);
        if (value.contains("failed") || value.contains("error") || value.contains("rejected")
                || value.contains("unavailable") || value.contains("lost") || value.contains("cancelled")) {
            return DANGER;
        }
        if (value.contains("warning") || value.contains("unencrypted") || value.contains("stale")
                || value.contains("reconnect") || value.contains("session only")) {
            return WARN;
        }
        if (value.contains("disconnected") || value.contains("not connected")) {
            return MUTED;
        }
        if (value.contains("completed") || value.contains("connected") || value.contains("saved")
                || value.contains("updated") || value.contains("added") || value.contains("opened")
                || value.contains("created") || value.contains("renamed") || value.contains("deleted")) {
            return SUCCESS;
        }
        return MUTED;
    }

    static int dp(Context context, int value) {
        return Math.round(value * context.getResources().getDisplayMetrics().density);
    }

    private static RippleDrawable ripple(Context context, int fill, int stroke, int rippleColor, int radiusDp) {
        GradientDrawable content = rounded(context, fill, stroke, radiusDp);
        return new RippleDrawable(ColorStateList.valueOf(rippleColor), content, null);
    }

    private static final class GhostArrayAdapter extends ArrayAdapter<String> {
        private final Context context;
        private final boolean spinner;

        GhostArrayAdapter(Context context, List<String> values, boolean spinner) {
            super(context, spinner ? android.R.layout.simple_spinner_item : android.R.layout.simple_list_item_1, values);
            this.context = context;
            this.spinner = spinner;
            if (spinner) setDropDownViewResource(android.R.layout.simple_spinner_dropdown_item);
        }

        @Override
        public View getView(int position, View convertView, ViewGroup parent) {
            return style(super.getView(position, convertView, parent), false);
        }

        @Override
        public View getDropDownView(int position, View convertView, ViewGroup parent) {
            return style(super.getDropDownView(position, convertView, parent), true);
        }

        private View style(View view, boolean dropDown) {
            if (!(view instanceof TextView)) return view;
            TextView text = (TextView) view;
            text.setTextColor(TEXT);
            text.setTextSize(spinner ? 14f : 13.5f);
            text.setGravity(Gravity.CENTER_VERTICAL);
            text.setMinHeight(dp(context, spinner ? 46 : 42));
            text.setPadding(dp(context, 12), dp(context, 7), dp(context, 12), dp(context, 7));
            text.setBackgroundColor(dropDown ? PANEL : Color.TRANSPARENT);
            return text;
        }
    }
}
