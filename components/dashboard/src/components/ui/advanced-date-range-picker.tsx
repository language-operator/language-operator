'use client'

import { useState, useRef, useEffect } from 'react'
import { CalendarIcon, ChevronLeftIcon, ChevronRightIcon, ClockIcon } from 'lucide-react'
import { format, addDays, addWeeks, addMonths, startOfDay, endOfDay, startOfWeek, endOfWeek, startOfMonth, endOfMonth, isSameDay, isWithinInterval } from 'date-fns'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

interface DateRange {
  from: Date
  to: Date
}

interface AdvancedDateRangePickerProps {
  date: DateRange
  onDateChange: (range: DateRange) => void
  className?: string
  onOpenChange?: (isOpen: boolean) => void
}

const PRESET_RANGES = [
  { label: 'Today', getValue: () => ({ from: startOfDay(new Date()), to: endOfDay(new Date()) }) },
  { label: 'Yesterday', getValue: () => ({ from: startOfDay(addDays(new Date(), -1)), to: endOfDay(addDays(new Date(), -1)) }) },
  { label: 'Last 24 Hours', getValue: () => ({ from: addDays(new Date(), -1), to: new Date() }) },
  { label: 'Last 2 Days', getValue: () => ({ from: addDays(new Date(), -2), to: new Date() }) },
  { label: 'Last 3 days', getValue: () => ({ from: addDays(new Date(), -3), to: new Date() }) },
  { label: 'Last 7 days', getValue: () => ({ from: addDays(new Date(), -7), to: new Date() }) },
  { label: 'Last 30 days', getValue: () => ({ from: addDays(new Date(), -30), to: new Date() }) },
  { label: 'Last 60 days', getValue: () => ({ from: addDays(new Date(), -60), to: new Date() }) },
  { label: 'Last 90 days', getValue: () => ({ from: addDays(new Date(), -90), to: new Date() }) },
  { label: 'Last 120 days', getValue: () => ({ from: addDays(new Date(), -120), to: new Date() }) },
  { label: 'This Week', getValue: () => ({ from: startOfWeek(new Date()), to: endOfWeek(new Date()) }) },
  { label: 'Last Week', getValue: () => ({ from: startOfWeek(addWeeks(new Date(), -1)), to: endOfWeek(addWeeks(new Date(), -1)) }) },
  { label: 'This Month', getValue: () => ({ from: startOfMonth(new Date()), to: endOfMonth(new Date()) }) },
  { label: 'Last Month', getValue: () => ({ from: startOfMonth(addMonths(new Date(), -1)), to: endOfMonth(addMonths(new Date(), -1)) }) },
]

const WEEKDAYS = ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa']
const MONTHS = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December'
]

export function AdvancedDateRangePicker({ date, onDateChange, className, onOpenChange }: AdvancedDateRangePickerProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [currentMonth, setCurrentMonth] = useState(new Date())
  const [selectingFrom, setSelectingFrom] = useState(true)
  const [tempRange, setTempRange] = useState<DateRange>(date)

  // Notify parent component when dropdown opens/closes
  useEffect(() => {
    onOpenChange?.(isOpen)
  }, [isOpen, onOpenChange])

  const handlePresetClick = (preset: typeof PRESET_RANGES[0]) => {
    const range = preset.getValue()
    setTempRange(range)
    setCurrentMonth(range.from) // Navigate calendar to the start date
    // Don't close the picker - let user review and apply manually
  }

  const handleApply = () => {
    onDateChange(tempRange)
    setIsOpen(false)
  }

  const handleDateClick = (day: Date) => {
    if (selectingFrom) {
      setTempRange({ ...tempRange, from: startOfDay(day) })
      setSelectingFrom(false)
    } else {
      if (day < tempRange.from) {
        setTempRange({ from: startOfDay(day), to: tempRange.from })
      } else {
        setTempRange({ ...tempRange, to: endOfDay(day) })
      }
      setSelectingFrom(true)
    }
  }

  const getDaysInMonth = (date: Date) => {
    const year = date.getFullYear()
    const month = date.getMonth()
    const firstDay = new Date(year, month, 1)
    const lastDay = new Date(year, month + 1, 0)
    const daysInMonth = lastDay.getDate()
    const startingDayOfWeek = firstDay.getDay()

    const days = []
    
    // Previous month's trailing days
    for (let i = 0; i < startingDayOfWeek; i++) {
      const day = new Date(year, month, -startingDayOfWeek + i + 1)
      days.push({ date: day, isCurrentMonth: false })
    }
    
    // Current month's days
    for (let i = 1; i <= daysInMonth; i++) {
      const day = new Date(year, month, i)
      days.push({ date: day, isCurrentMonth: true })
    }
    
    // Next month's leading days
    const remainingDays = 42 - days.length // 6 rows * 7 days
    for (let i = 1; i <= remainingDays; i++) {
      const day = new Date(year, month + 1, i)
      days.push({ date: day, isCurrentMonth: false })
    }
    
    return days
  }

  const isDateInRange = (day: Date) => {
    if (!tempRange.from || !tempRange.to) return false
    return isWithinInterval(day, { start: tempRange.from, end: tempRange.to })
  }

  const isDateSelected = (day: Date) => {
    return isSameDay(day, tempRange.from) || isSameDay(day, tempRange.to)
  }

  const days = getDaysInMonth(currentMonth)

  return (
    <div className={cn('space-y-4 w-full', className)}>
      <Button
        variant="outline"
        className="w-full justify-center text-center font-normal"
        onClick={() => setIsOpen(!isOpen)}
      >
        <CalendarIcon className="mr-2 h-4 w-4" />
        {format(date.from, 'MMM dd, yyyy')} - {format(date.to, 'MMM dd, yyyy')}
      </Button>

      {isOpen && (
        <div className="w-full py-8 relative">
          <div className="flex w-full gap-12">
            {/* Calendar */}
            <div className="flex-1 space-y-6">
              {/* Calendar Header */}
              <div className="flex items-center justify-between">
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-10 w-10 rounded-lg hover:bg-background/60"
                  onClick={() => setCurrentMonth(addMonths(currentMonth, -1))}
                >
                  <ChevronLeftIcon className="h-5 w-5" />
                </Button>
                <div className="font-normal text-xl text-foreground/90 tracking-wide uppercase">
                  {MONTHS[currentMonth.getMonth()]} {currentMonth.getFullYear()}
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-10 w-10 rounded-lg hover:bg-background/60"
                  onClick={() => setCurrentMonth(addMonths(currentMonth, 1))}
                >
                  <ChevronRightIcon className="h-5 w-5" />
                </Button>
              </div>

              {/* Weekday Headers */}
              <div className="grid grid-cols-7 gap-2">
                {WEEKDAYS.map((day) => (
                  <div key={day} className="text-center text-sm font-medium text-muted-foreground/70 p-3">
                    {day}
                  </div>
                ))}
              </div>

              {/* Calendar Days */}
              <div className="grid grid-cols-7 gap-2">
                {days.map((day, index) => (
                  <Button
                    key={index}
                    variant="ghost"
                    className={cn(
                      'h-12 w-full p-0 text-sm font-light rounded-lg transition-all duration-200',
                      !day.isCurrentMonth && 'text-muted-foreground/40 hover:text-muted-foreground/60',
                      day.isCurrentMonth && 'text-foreground/80 hover:text-foreground hover:bg-background/60',
                      isDateSelected(day.date) && 'bg-primary/20 text-primary border border-primary/30',
                      isDateInRange(day.date) && !isDateSelected(day.date) && 'bg-primary/5 border border-primary/10',
                      isSameDay(day.date, new Date()) && 'ring-2 ring-primary/30'
                    )}
                    onClick={() => handleDateClick(day.date)}
                  >
                    {day.date.getDate()}
                  </Button>
                ))}
              </div>

            </div>

            {/* Presets */}
            <div className="flex-1 space-y-6">
              <div className="flex flex-wrap gap-3">
                {PRESET_RANGES.map((preset, index) => (
                  <Badge
                    key={index}
                    variant="outline"
                    className="cursor-pointer px-4 py-2 text-sm font-light hover:bg-background/60 transition-all duration-200 text-foreground/80 hover:text-foreground"
                    onClick={() => handlePresetClick(preset)}
                  >
                    {preset.label}
                  </Badge>
                ))}
              </div>
              
              {/* Selected Range Display */}
              <div className="space-y-2 pt-4 border-t border-border/20">
                <div className="text-sm font-bold text-muted-foreground/70 tracking-wide">
                  Selected range:
                </div>
                <div className="text-foreground/90 font-light tracking-wide">
                  {format(tempRange.from, 'MMMM do, yyyy')} to {format(tempRange.to, 'MMMM do, yyyy')}
                </div>
              </div>
            </div>
          </div>
          
          {/* Action Buttons - Fixed to bottom right */}
          <div className="absolute bottom-0 right-0 flex gap-3">
            <Button
              variant="outline"
              onClick={() => setIsOpen(false)}
            >
              Cancel
            </Button>
            <Button
              variant="default"
              onClick={handleApply}
            >
              Apply Range
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}