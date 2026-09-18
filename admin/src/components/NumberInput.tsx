import React, { useEffect, useState } from 'react';

interface NumberInputProps {
  value: number;
  onChange: (value: number) => void;
  min?: number;
  max?: number;
  step?: number;
  suffix?: string;
  label?: string;
  id?: string;
}

const clamp = (value: number, min?: number, max?: number) => {
  let next = value;
  if (typeof min === 'number') {
    next = Math.max(min, next);
  }
  if (typeof max === 'number') {
    next = Math.min(max, next);
  }
  return next;
};

export const NumberInput: React.FC<NumberInputProps> = ({
  value,
  onChange,
  min,
  max,
  step = 0.05,
  suffix,
  id,
}) => {
  const [draft, setDraft] = useState(String(Number.isFinite(value) ? value : 0));
  const [focused, setFocused] = useState(false);

  useEffect(() => {
    if (!focused) {
      setDraft(String(Number.isFinite(value) ? value : 0));
    }
  }, [focused, value]);

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const raw = event.target.value;
    setDraft(raw);

    if (raw === '' || raw === '-' || raw === '+' || raw === '.' || raw === '-.' || raw === '+.') {
      return;
    }

    const parsed = Number(raw);
    if (Number.isFinite(parsed)) {
      onChange(clamp(parsed, min, max));
    }
  };

  const handleBlur = () => {
    setFocused(false);

    const parsed = Number(draft);
    const next = Number.isFinite(parsed) ? clamp(parsed, min, max) : clamp(0, min, max);
    setDraft(String(next));
    onChange(next);
  };

  return (
    <div className="relative">
      <input
        id={id}
        type="number"
        inputMode="decimal"
        step={step}
        min={min}
        max={max}
        value={draft}
        onFocus={() => setFocused(true)}
        onChange={handleChange}
        onBlur={handleBlur}
        className="block w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm font-medium text-gray-900 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-200"
      />
      {suffix && (
        <span className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-xs text-gray-500">
          {suffix}
        </span>
      )}
    </div>
  );
};
