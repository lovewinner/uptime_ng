export type ExportChoice = 'all' | 'selected'

export interface ExportMonitorOption {
  id: number
  name: string
}

export function selectedExportIDs(monitors: ExportMonitorOption[]): number[] {
  return monitors.filter((monitor) => monitor.name !== '').map((monitor) => monitor.id)
}

export function exportIDsForChoice(choice: ExportChoice, monitors: ExportMonitorOption[]): number[] | undefined {
  return choice === 'selected' ? selectedExportIDs(monitors) : undefined
}
