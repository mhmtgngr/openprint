# Check virtual printers
Write-Host "=== OpenPrint Virtual Printers ==="
Get-Printer | Where-Object { $_.Name -like 'OpenPrint*' } | Format-Table Name, DriverName, PortName, PrinterStatus -AutoSize

# Check event log
Write-Host "`n=== Recent Agent Events ==="
Get-EventLog -LogName Application -Source 'OpenPrintAgent' -Newest 10 -ErrorAction SilentlyContinue | Format-List TimeGenerated, EntryType, Message

# Check TCP listeners
Write-Host "`n=== TCP Listeners on 9100/9101 ==="
netstat -an | Select-String "9100|9101"
