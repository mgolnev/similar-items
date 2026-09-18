/** Человекочитаемые названия стадий отбора (thresholds.*.strategy / preview result.strategy). */
const STRATEGY_LABELS: Record<string, string> = {
  same_category_color_then_parent_fallback:
    'Сначала та же категория и цвет, при нехватке — родительская категория',
  parent_category_fallback: 'Запасной отбор по родительской категории',
  branch_fallback: 'Запасной отбор по ветке каталога',
};

export function strategyStageLabel(strategy: string | null | undefined): string {
  if (!strategy) return '';
  return STRATEGY_LABELS[strategy] || strategy;
}
