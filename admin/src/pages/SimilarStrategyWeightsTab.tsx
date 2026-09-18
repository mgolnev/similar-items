import React from 'react';
import { NumberInput } from '../components/NumberInput';
import { SimilarStrategyDocument } from '../types/strategy';

type Props = {
  doc: SimilarStrategyDocument;
  onChange: (doc: SimilarStrategyDocument) => void;
};

type CategoryTierKey =
  | 'category_leaf'
  | 'category_shared_leaf'
  | 'category_parent_deep'
  | 'category_parent'
  | 'category_parent_shallow'
  | 'category_branch';

/** Живой пример для блока «Категория — лестница» (данные из prod-фида). */
const CATEGORY_LADDER_WALKTHROUGH = {
  baseSku: 'GOW004417-2',
  baseName: 'ветровка Kuromi для девочки',
  baseLeaf: 'Ветровки для девочек',
  basePath: 'Дети → Девочки → … → Куртки для девочек → Ветровки для девочек',
} as const;

const CATEGORY_TIERS: Array<{
  key: CategoryTierKey;
  label: string;
  hint: string;
}> = [
  {
    key: 'category_leaf',
    label: 'Листовая категория',
    hint:
      `Пример: база ${CATEGORY_LADDER_WALKTHROUGH.baseSku} (${CATEGORY_LADDER_WALKTHROUGH.baseLeaf}) → другая ветровка в том же leaf «Ветровки для девочек». Самый сильный сигнал «тот же тип товара».`,
  },
  {
    key: 'category_shared_leaf',
    label: 'Общий leaf (любая ветка фида)',
    hint:
      'Пример: у SKU две ветки в фиде; leaf совпал не в основной категории, а во второй ветке «Каталог». Реже leaf, но та же конечная категория.',
  },
  {
    key: 'category_parent_deep',
    label: 'Родитель deep (depth ≥ 3)',
    hint:
      `Пример: ${CATEGORY_LADDER_WALKTHROUGH.baseSku} (ветровка) → куртка/пуховик в leaf «Куртки для девочек». Общий узел «Куртки для девочек», глубина 3 — похожая верхняя одежда, но не тот же leaf.`,
  },
  {
    key: 'category_parent',
    label: 'Родитель mid (depth = 2)',
    hint:
      'Пример: GAS018091-1 (косынка) → другая косынка или шапка. Общий узел «Женские головные уборы», глубина 2 — один класс аксессуаров.',
  },
  {
    key: 'category_parent_shallow',
    label: 'Родитель shallow (depth = 1)',
    hint:
      `Пример: ${CATEGORY_LADDER_WALKTHROUGH.baseSku} → GSK020555-1 (джинсовая юбка). Общий только «Девочки», глубина 1 — разный тип (ветровка vs юбка); вес должен быть низким.`,
  },
  {
    key: 'category_branch',
    label: 'Ветка каталога',
    hint:
      'Пример: женское платье и женские брюки — совпал только корень «Женщины», без общего родителя ниже. Слабый сигнал, ниже parent shallow.',
  },
];

const ADDITIVE_SIGNALS: Array<{
  key: 'article_prefix';
  label: string;
  compares: string;
}> = [
  { key: 'article_prefix', label: 'Префикс артикула', compares: 'первые 3 буквы group_id' },
];

export const SimilarStrategyWeightsTab: React.FC<Props> = ({ doc, onChange }) => {
  const setWeight = (key: keyof SimilarStrategyDocument['weights'], v: number) =>
    onChange({ ...doc, weights: { ...doc.weights, [key]: v } });

  return (
    <div className="space-y-6">
      {/* Каскад категории */}
      <section>
        <div className="mb-2 flex items-baseline justify-between gap-2">
          <h3 className="text-sm font-semibold text-slate-800">Категория — лестница</h3>
          <span className="rounded-full bg-amber-100 px-2 py-0.5 text-[11px] text-amber-800">
            срабатывает только ОДИН уровень (первое совпадение сверху вниз)
          </span>
        </div>
        <p className="mb-2 text-xs text-slate-500">
          Баллы за категорию не складываются: движок идёт сверху вниз и берёт первый совпавший
          уровень. Поэтому верхние уровни обычно дороже нижних.
        </p>
        <div className="mb-3 rounded-lg border border-violet-100 bg-violet-50/80 px-3 py-2.5 text-xs text-violet-900">
          <span className="font-semibold">Сквозной пример — база {CATEGORY_LADDER_WALKTHROUGH.baseSku}</span>
          {' '}
          ({CATEGORY_LADDER_WALKTHROUGH.baseName}): путь{' '}
          <span className="font-mono text-[11px]">{CATEGORY_LADDER_WALKTHROUGH.basePath}</span>.
          {' '}Другая ветровка → уровень 1; куртка в «Куртки для девочек» → 3; юбка GSK020555-1 →
          только «Девочки» (уровень 5). Если category = 0 у всех — проверьте путь в фиде
          (сломанный пример: GPT011623-1 с путём из одного узла «Брюки для девушек»).
        </div>
        <div className="overflow-hidden rounded-lg border border-slate-200">
          {CATEGORY_TIERS.map((tier, i) => (
            <div
              key={tier.key}
              className="flex items-center gap-3 border-b border-slate-100 px-3 py-2.5 last:border-0"
            >
              <div className="flex w-6 shrink-0 flex-col items-center text-slate-300">
                <span className="text-xs font-semibold text-violet-500">{i + 1}</span>
                {i < CATEGORY_TIERS.length - 1 && <span className="text-[10px]">↓</span>}
              </div>
              <div className="min-w-0 flex-1">
                <div className="text-sm font-medium text-slate-800">{tier.label}</div>
                <div className="text-xs text-slate-500">{tier.hint}</div>
              </div>
              <div className="w-28 shrink-0">
                <NumberInput
                  value={doc.weights[tier.key]}
                  min={0}
                  max={200}
                  step={1}
                  onChange={(v) => setWeight(tier.key, v)}
                />
              </div>
            </div>
          ))}
        </div>
        {!isDescending(doc) && (
          <p className="mt-2 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800">
            Веса лестницы не убывают сверху вниз. Обычно лист должен быть дороже ветки, иначе слабое
            совпадение даёт столько же баллов, сколько точное.
          </p>
        )}
      </section>

      {/* Аддитивные сигналы */}
      <section>
        <h3 className="mb-2 text-sm font-semibold text-slate-800">Аддитивные сигналы</h3>
        <p className="mb-3 text-xs text-slate-500">
          Эти баллы складываются с категорией при каждом совпадении. Цвет, тип и прочие атрибуты
          настраиваются во вкладке «Правила атрибутов».
        </p>
        <div className="overflow-hidden rounded-lg border border-slate-200">
          <div className="grid grid-cols-[1fr_auto] items-center gap-2 bg-slate-50 px-3 py-1.5 text-[11px] font-medium uppercase tracking-wide text-slate-500">
            <span>Сигнал</span>
            <span className="text-right">Балл</span>
          </div>
          {ADDITIVE_SIGNALS.map((s) => (
            <div
              key={s.key}
              className="grid grid-cols-[1fr_auto] items-center gap-2 border-t border-slate-100 px-3 py-2.5"
            >
              <div>
                <div className="text-sm text-slate-800">{s.label}</div>
                <div className="font-mono text-[11px] text-slate-400">{s.compares}</div>
              </div>
              <div className="w-28">
                <NumberInput
                  value={doc.weights[s.key]}
                  min={0}
                  max={200}
                  step={1}
                  onChange={(v) => setWeight(s.key, v)}
                />
              </div>
            </div>
          ))}
          {/* Цена: вес + ширина коридора */}
          <div className="grid grid-cols-[1fr_auto] items-center gap-2 border-t border-slate-100 px-3 py-2.5">
            <div>
              <div className="text-sm text-slate-800">Цена в коридоре</div>
              <div className="font-mono text-[11px] text-slate-400">
                |цена − база| ≤ ±{doc.price.range_percent}%
              </div>
            </div>
            <div className="w-28">
              <NumberInput
                value={doc.weights.price}
                min={0}
                max={200}
                step={1}
                onChange={(v) => setWeight('price', v)}
              />
            </div>
          </div>
          <div className="grid grid-cols-[1fr_auto] items-center gap-2 border-t border-slate-100 bg-slate-50/60 px-3 py-2.5">
            <div className="text-xs text-slate-500">Ширина ценового коридора, ±%</div>
            <div className="w-28">
              <NumberInput
                value={doc.price.range_percent}
                min={5}
                max={80}
                step={5}
                onChange={(v) => onChange({ ...doc, price: { range_percent: v } })}
              />
            </div>
          </div>
        </div>
      </section>
    </div>
  );
};

function isDescending(doc: SimilarStrategyDocument): boolean {
  const w = doc.weights;
  return (
    w.category_leaf >= w.category_shared_leaf &&
    w.category_shared_leaf >= w.category_parent_deep &&
    w.category_parent_deep >= w.category_parent &&
    w.category_parent >= w.category_parent_shallow &&
    w.category_parent_shallow >= w.category_branch
  );
}
