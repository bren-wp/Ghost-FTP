using Microsoft.Win32;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Media;

namespace GhostFTP.UI;

internal static class Theme
{
    public static readonly FontFamily UiFont = new("Segoe UI Variable Text, Segoe UI");
    public static readonly FontFamily DisplayFont = new("Segoe UI Variable Display, Segoe UI");

    public static bool IsSystemDark()
    {
        try
        {
            using var key = Registry.CurrentUser.OpenSubKey(@"Software\Microsoft\Windows\CurrentVersion\Themes\Personalize");
            return key?.GetValue("AppsUseLightTheme") is int value && value == 0;
        }
        catch { return true; }
    }

    public static void Apply(bool dark)
    {
        var resources = Application.Current.Resources;
        resources["Bg"] = Brush(dark ? "#0B0D12" : "#F4F6FA");
        resources["Surface"] = Brush(dark ? "#11151D" : "#FFFFFF");
        resources["Surface2"] = Brush(dark ? "#171C26" : "#F7F9FC");
        resources["SurfaceHover"] = Brush(dark ? "#202735" : "#EAF0FA");
        resources["Text"] = Brush(dark ? "#F5F7FB" : "#161A22");
        resources["Muted"] = Brush(dark ? "#9BA6B7" : "#667085");
        resources["Border"] = Brush(dark ? "#2A3242" : "#D8DEE9");
        resources["Accent"] = Brush("#7C5CFF");
        resources["AccentHover"] = Brush("#8D73FF");
        resources["AccentSoft"] = Brush(dark ? "#2A214D" : "#EEE9FF");
        resources["Success"] = Brush("#38C793");
        resources["Danger"] = Brush("#FF5D6C");
        resources["Warning"] = Brush("#F0B44D");
    }

    public static Brush R(string key) => (Brush)Application.Current.Resources[key];
    private static SolidColorBrush Brush(string hex)
    {
        var brush = new SolidColorBrush((Color)ColorConverter.ConvertFromString(hex));
        brush.Freeze();
        return brush;
    }

    public static Border Card(UIElement child, Thickness? padding = null)
    {
        return new Border
        {
            Background = R("Surface"),
            BorderBrush = R("Border"),
            BorderThickness = new Thickness(1),
            CornerRadius = new CornerRadius(14),
            Padding = padding ?? new Thickness(16),
            Child = child
        };
    }

    public static TextBlock Text(string text, double size = 13, bool muted = false, FontWeight? weight = null)
    {
        return new TextBlock
        {
            Text = text,
            Foreground = R(muted ? "Muted" : "Text"),
            FontFamily = UiFont,
            FontSize = size,
            FontWeight = weight ?? FontWeights.Normal,
            TextWrapping = TextWrapping.Wrap,
            VerticalAlignment = VerticalAlignment.Center
        };
    }

    public static Button Button(string text, bool primary = false, bool danger = false)
    {
        var button = new Button
        {
            Content = text,
            FontFamily = UiFont,
            FontSize = 13,
            FontWeight = FontWeights.SemiBold,
            Foreground = danger ? Brushes.White : primary ? Brushes.White : R("Text"),
            Background = danger ? R("Danger") : primary ? R("Accent") : R("Surface2"),
            BorderBrush = danger ? R("Danger") : primary ? R("Accent") : R("Border"),
            BorderThickness = new Thickness(1),
            Padding = new Thickness(14, 8, 14, 8),
            MinHeight = 36,
            Cursor = System.Windows.Input.Cursors.Hand,
            Template = RoundedButtonTemplate()
        };
        return button;
    }

    public static TextBox TextBox(string? text = null)
    {
        return new TextBox
        {
            Text = text ?? string.Empty,
            FontFamily = UiFont,
            FontSize = 13,
            Foreground = R("Text"),
            Background = R("Surface2"),
            BorderBrush = R("Border"),
            BorderThickness = new Thickness(1),
            Padding = new Thickness(10, 8, 10, 8),
            MinHeight = 36,
            CaretBrush = R("Text"),
            Template = RoundedTextBoxTemplate()
        };
    }

    public static PasswordBox PasswordBox()
    {
        return new PasswordBox
        {
            FontFamily = UiFont,
            FontSize = 13,
            Foreground = R("Text"),
            Background = R("Surface2"),
            BorderBrush = R("Border"),
            BorderThickness = new Thickness(1),
            Padding = new Thickness(10, 8, 10, 8),
            MinHeight = 36,
            CaretBrush = R("Text")
        };
    }

    public static ComboBox ComboBox()
    {
        return new ComboBox
        {
            FontFamily = UiFont,
            FontSize = 13,
            Foreground = R("Text"),
            Background = R("Surface2"),
            BorderBrush = R("Border"),
            BorderThickness = new Thickness(1),
            Padding = new Thickness(8, 6, 8, 6),
            MinHeight = 36
        };
    }

    private static ControlTemplate RoundedButtonTemplate()
    {
#pragma warning disable CS0618
        var border = new FrameworkElementFactory(typeof(Border));
        border.SetBinding(Border.BackgroundProperty, new System.Windows.Data.Binding("Background") { RelativeSource = new System.Windows.Data.RelativeSource(System.Windows.Data.RelativeSourceMode.TemplatedParent) });
        border.SetBinding(Border.BorderBrushProperty, new System.Windows.Data.Binding("BorderBrush") { RelativeSource = new System.Windows.Data.RelativeSource(System.Windows.Data.RelativeSourceMode.TemplatedParent) });
        border.SetBinding(Border.BorderThicknessProperty, new System.Windows.Data.Binding("BorderThickness") { RelativeSource = new System.Windows.Data.RelativeSource(System.Windows.Data.RelativeSourceMode.TemplatedParent) });
        border.SetValue(Border.CornerRadiusProperty, new CornerRadius(9));
        border.SetValue(Border.SnapsToDevicePixelsProperty, true);
        var presenter = new FrameworkElementFactory(typeof(ContentPresenter));
        presenter.SetValue(ContentPresenter.HorizontalAlignmentProperty, HorizontalAlignment.Center);
        presenter.SetValue(ContentPresenter.VerticalAlignmentProperty, VerticalAlignment.Center);
        presenter.SetBinding(ContentPresenter.MarginProperty, new System.Windows.Data.Binding("Padding") { RelativeSource = new System.Windows.Data.RelativeSource(System.Windows.Data.RelativeSourceMode.TemplatedParent) });
        border.AppendChild(presenter);
        var template = new ControlTemplate(typeof(Button)) { VisualTree = border };
        var hover = new Trigger { Property = UIElement.IsMouseOverProperty, Value = true };
        hover.Setters.Add(new Setter(Control.OpacityProperty, 0.88));
        template.Triggers.Add(hover);
        var pressed = new Trigger { Property = Button.IsPressedProperty, Value = true };
        pressed.Setters.Add(new Setter(Control.OpacityProperty, 0.72));
        template.Triggers.Add(pressed);
        var disabled = new Trigger { Property = UIElement.IsEnabledProperty, Value = false };
        disabled.Setters.Add(new Setter(Control.OpacityProperty, 0.45));
        template.Triggers.Add(disabled);
        return template;
#pragma warning restore CS0618
    }

    private static ControlTemplate RoundedTextBoxTemplate()
    {
#pragma warning disable CS0618
        var border = new FrameworkElementFactory(typeof(Border));
        border.SetBinding(Border.BackgroundProperty, new System.Windows.Data.Binding("Background") { RelativeSource = new System.Windows.Data.RelativeSource(System.Windows.Data.RelativeSourceMode.TemplatedParent) });
        border.SetBinding(Border.BorderBrushProperty, new System.Windows.Data.Binding("BorderBrush") { RelativeSource = new System.Windows.Data.RelativeSource(System.Windows.Data.RelativeSourceMode.TemplatedParent) });
        border.SetBinding(Border.BorderThicknessProperty, new System.Windows.Data.Binding("BorderThickness") { RelativeSource = new System.Windows.Data.RelativeSource(System.Windows.Data.RelativeSourceMode.TemplatedParent) });
        border.SetValue(Border.CornerRadiusProperty, new CornerRadius(9));
        var host = new FrameworkElementFactory(typeof(ScrollViewer));
        host.SetValue(FrameworkElement.NameProperty, "PART_ContentHost");
        host.SetBinding(FrameworkElement.MarginProperty, new System.Windows.Data.Binding("Padding") { RelativeSource = new System.Windows.Data.RelativeSource(System.Windows.Data.RelativeSourceMode.TemplatedParent) });
        border.AppendChild(host);
        return new ControlTemplate(typeof(TextBox)) { VisualTree = border };
#pragma warning restore CS0618
    }
}
