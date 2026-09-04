using GhostFTP.Core.Models;
using GhostFTP.Core.Protocol;
using GhostFTP.Core.Services;
using GhostFTP.Services;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Media;


namespace GhostFTP.UI;

public sealed partial class MainWindow
{
    private Button ToolButton(string text, Action action)
    {
        var button = Theme.Button(text);
        button.Padding = new Thickness(11, 6, 11, 6);
        button.MinHeight = 32;
        button.Click += (_, _) => action();
        return button;
    }

    private Button ToolButton(string text, Func<Task> action)
    {
        var button = Theme.Button(text);
        button.Padding = new Thickness(11, 6, 11, 6);
        button.MinHeight = 32;
        button.Click += async (_, _) =>
        {
            try { await action(); }
            catch (Exception ex) { ShowOperationError("The operation failed.", ex); }
        };
        return button;
    }

    private static GridView CreateFileGrid(bool local)
    {
        var grid = new GridView { AllowsColumnReorder = true };
        grid.Columns.Add(Column("Name", "Name", 250));
        grid.Columns.Add(Column("Type", "Type", 80));
        grid.Columns.Add(Column("Size", "SizeText", 100));
        grid.Columns.Add(Column("Modified", "ModifiedText", 145));
        if (!local) grid.Columns.Add(Column("Permissions", "Permissions", 110));
        return grid;
    }

    private static GridView CreateQueueGrid()
    {
        var grid = new GridView();
        grid.Columns.Add(Column("Item", "DisplayName", 180));
        grid.Columns.Add(Column("Direction", "Direction", 90));
        grid.Columns.Add(Column("State", "State", 90));
        grid.Columns.Add(Column("Progress", "ProgressText", 80));
        grid.Columns.Add(Column("Speed", "SpeedText", 100));
        grid.Columns.Add(Column("Source", "Source", 330));
        grid.Columns.Add(Column("Destination", "Destination", 330));
        return grid;
    }

    private static GridViewColumn Column(string title, string binding, double width) => new()
    {
        Header = title,
        Width = width,
        DisplayMemberBinding = new System.Windows.Data.Binding(binding)
    };

    private static ContextMenu CreateContextMenu(params (string text, RoutedEventHandler handler)[] items)
    {
        var menu = new ContextMenu();
        foreach (var item in items)
        {
            var menuItem = new MenuItem { Header = item.text };
            menuItem.Click += item.handler;
            menu.Items.Add(menuItem);
        }
        return menu;
    }

    private static void AddAt(Grid grid, UIElement element, int column)
    {
        Grid.SetColumn(element, column);
        grid.Children.Add(element);
    }

    private void ShowOperationError(string message, Exception ex) =>
        MessageBox.Show(this, message + "\n\n" + ex.Message, "GhostFTP", MessageBoxButton.OK, MessageBoxImage.Error);

    private static string FormatBytes(long value)
    {
        double number = Math.Max(0, value);
        string[] units = ["B", "KB", "MB", "GB", "TB"];
        var index = 0;
        while (number >= 1024 && index < units.Length - 1) { number /= 1024; index++; }
        return $"{number:0.#} {units[index]}";
    }
}
