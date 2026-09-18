import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import toast from 'react-hot-toast';
import {
  ArrowDownTrayIcon,
  ArrowLeftIcon,
  ArrowUpTrayIcon,
  CheckIcon,
  PlusIcon,
  SparklesIcon,
  SquaresPlusIcon,
  StarIcon,
  TrashIcon,
} from '@heroicons/react/24/outline';
import axios from 'axios';
import { NumberInput } from '../components/NumberInput';
import {
  strategyApi,
  StrategyPresetSummary,
} from '../services/strategyApi';
import { SimilarPreviewResult, SimilarStrategyDocument } from '../types/strategy';
import { FeatureCatalogEntry } from '../types/featureCatalog';
import { SimilarStrategyPreviewTab } from './SimilarStrategyPreviewTab';
import { SimilarStrategyTitlePreviewTab } from './SimilarStrategyTitlePreviewTab';
import { SimilarStrategyColorPreviewTab } from './SimilarStrategyColorPreviewTab';
import { SimilarStrategyColorsTab } from './SimilarStrategyColorsTab';
import { SimilarStrategyRulesTab } from './SimilarStrategyRulesTab';
import { SimilarStrategySynonymsTab } from './SimilarStrategySynonymsTab';
import { SimilarStrategyTitlePlayground } from './SimilarStrategyTitlePlayground';
import { SimilarStrategyVisualTab } from './SimilarStrategyVisualTab';
import { SimilarStrategyWeightsTab } from './SimilarStrategyWeightsTab';
import { ThresholdCascadeExplainer } from '../components/ThresholdCascadeExplainer';
import { strategyStageLabel } from '../utils/strategyStageLabel';
import { downloadPresetExport, readPresetImportFile } from '../utils/presetTransfer';
import { VisualPreviewRow } from '../services/strategyApi';
import { RecommendationCompositionPanel } from '../components/RecommendationCompositionPanel';

type TabId = 'signals' | 'selection' | 'showcase' | 'preview';
type SignalSection = 'weights' | 'rules' | 'colors' | 'title' | 'synonyms';
type PreviewMode = 'products' | 'title' | 'colors' | 'visual';

export const SimilarStrategy: React.FC = () => {
  const [tab, setTab] = useState<TabId>('signals');
  const [signalSection, setSignalSection] = useState<SignalSection>('weights');
  const [previewMode, setPreviewMode] = useState<PreviewMode>('products');
  const [doc, setDoc] = useState<SimilarStrategyDocument | null>(null);
  const [saveReason, setSaveReason] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [previewSection, setPreviewSection] = useState('all');
  const [previewProductId, setPreviewProductId] = useState('');
  const [previewLimit, setPreviewLimit] = useState(12);
  const [previewResult, setPreviewResult] = useState<SimilarPreviewResult | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [loadError, setLoadError] = useState('');
  const [featureCatalog, setFeatureCatalog] = useState<FeatureCatalogEntry[]>([]);
  const [catalogLoading, setCatalogLoading] = useState(false);
  const [presets, setPresets] = useState<StrategyPresetSummary[]>([]);
  const [activePresetId, setActivePresetId] = useState('');
  const [presetName, setPresetName] = useState('');
  const [productionId, setProductionId] = useState('');
  const [builtinDefault, setBuiltinDefault] = useState<SimilarStrategyDocument | null>(null);
  const [auditMaxAvg, setAuditMaxAvg] = useState<number | ''>(70);
  const [auditLoading, setAuditLoading] = useState(false);
  const [visualRows, setVisualRows] = useState<VisualPreviewRow[]>([]);
  const [visualLoading, setVisualLoading] = useState(false);
  const importInputRef = useRef<HTMLInputElement>(null);

  const loadPreset = useCallback(async (id: string) => {
    const view = await strategyApi.getPreset(id);
    setDoc(view.strategy);
    setPresetName(view.name);
    setActivePresetId(view.id);
  }, []);

  const refreshPresets = useCallback(async (selectId?: string) => {
    const list = await strategyApi.listPresets();
    setPresets(list.presets ?? []);
    setProductionId(list.production_id ?? '');
    const nextId = selectId || list.production_id || list.presets?.[0]?.id || '';
    if (nextId) {
      await loadPreset(nextId);
    }
  }, [loadPreset]);

  const load = useCallback(async () => {
    setLoading(true);
    setLoadError('');
    try {
      const data = await strategyApi.getStrategy();
      setBuiltinDefault(data.builtin_default);
      await refreshPresets(data.production_id);
    } catch (e: unknown) {
      console.error(e);
      const msg =
        axios.isAxiosError(e) && e.response?.status === 404
          ? 'API не найден — перезапустите бэкенд'
          : 'Не удалось загрузить стратегию. Проверьте, что бэкенд запущен';
      setLoadError(msg);
      toast.error(msg);
      setDoc(null);
    } finally {
      setLoading(false);
    }
  }, [refreshPresets]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    if (!(tab === 'signals' && signalSection === 'rules')) return;
    let cancelled = false;
    (async () => {
      setCatalogLoading(true);
      try {
        const data = await strategyApi.getFeatureCatalog();
        if (!cancelled) setFeatureCatalog(data.features || []);
      } catch {
        if (!cancelled) setFeatureCatalog([]);
      } finally {
        if (!cancelled) setCatalogLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [tab, signalSection]);

  const handleSave = async () => {
    if (!doc || !activePresetId) return;
    setSaving(true);
    try {
      await strategyApi.updatePreset(activePresetId, presetName.trim() || 'Без названия', doc, saveReason);
      toast.success('Пресет сохранён');
      await refreshPresets(activePresetId);
    } catch {
      toast.error('Ошибка сохранения');
    } finally {
      setSaving(false);
    }
  };

  const handleLoadDefault = () => {
    if (!builtinDefault) return;
    setDoc(JSON.parse(JSON.stringify(builtinDefault)) as SimilarStrategyDocument);
    toast.success('Загружены встроенные дефолты — сохраните пресет, чтобы зафиксировать');
  };

  const handlePromote = async () => {
    if (!activePresetId) return;
    if (
      !window.confirm(
        'Запустится тяжёлый процесс прогрева кэша: пресет сохранится, станет продакшн, ' +
          'и движок пересчитает рекомендации под новые настройки. ' +
          'Данные публичного API обновятся в течение нескольких минут (до завершения отдаются текущие). Продолжить?'
      )
    ) {
      return;
    }
    setSaving(true);
    try {
      await strategyApi.updatePreset(
        activePresetId,
        presetName.trim() || 'Без названия',
        doc!,
        saveReason
      );
      await strategyApi.promotePreset(activePresetId);
      toast.success(
        'Прогрев запущен — рекомендации обновятся в течение нескольких минут',
        { duration: 6000 }
      );
      await refreshPresets(activePresetId);
    } catch {
      toast.error('Ошибка запуска прогрева');
    } finally {
      setSaving(false);
    }
  };

  const handleCreatePreset = async () => {
    if (!doc) return;
    const name = window.prompt('Имя нового пресета', 'Черновик');
    if (!name?.trim()) return;
    setSaving(true);
    try {
      const created = await strategyApi.createPreset(name.trim(), doc);
      toast.success('Пресет создан');
      await refreshPresets(created.id);
    } catch {
      toast.error('Не удалось создать пресет');
    } finally {
      setSaving(false);
    }
  };

  const handleDeletePreset = async () => {
    if (!activePresetId) return;
    if (activePresetId === productionId) {
      toast.error('Нельзя удалить продакшн-пресет');
      return;
    }
    if (!window.confirm(`Удалить пресет «${presetName}»?`)) return;
    setSaving(true);
    try {
      await strategyApi.deletePreset(activePresetId);
      toast.success('Пресет удалён');
      await refreshPresets(productionId);
    } catch {
      toast.error('Ошибка удаления');
    } finally {
      setSaving(false);
    }
  };

  const handleSelectPreset = async (id: string) => {
    if (id === activePresetId) return;
    setSaving(true);
    try {
      await loadPreset(id);
    } catch {
      toast.error('Не удалось загрузить пресет');
    } finally {
      setSaving(false);
    }
  };

  const handleExportPreset = () => {
    if (!doc) return;
    const name = presetName.trim() || 'Без названия';
    downloadPresetExport(name, doc);
    toast.success(`Пресет «${name}» экспортирован`);
  };

  const handleImportPresetClick = () => {
    importInputRef.current?.click();
  };

  const handleImportPresetFile = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;

    setSaving(true);
    try {
      const imported = await readPresetImportFile(file);
      const name =
        window.prompt('Имя пресета после импорта', imported.name)?.trim() || imported.name.trim();
      if (!name) {
        toast.error('Импорт отменён — не указано имя');
        return;
      }

      const createNew = window.confirm(
        `Импорт «${name}».\n\nOK — создать новый пресет\nОтмена — ${
          activePresetId ? `заменить текущий «${presetName}»` : 'отменить импорт'
        }`
      );

      if (createNew) {
        const created = await strategyApi.createPreset(name, imported.strategy);
        toast.success(`Пресет «${name}» импортирован`);
        await refreshPresets(created.id);
        return;
      }

      if (!activePresetId) {
        return;
      }

      if (
        !window.confirm(
          `Заменить настройки текущего пресета «${presetName}» импортом «${name}»? Это действие можно отменить повторным импортом бэкапа.`
        )
      ) {
        return;
      }

      await strategyApi.updatePreset(
        activePresetId,
        presetName.trim() || name,
        imported.strategy,
        saveReason.trim() || `import ${file.name}`
      );
      setDoc(imported.strategy);
      toast.success(`Пресет «${presetName}» обновлён из файла`);
      await refreshPresets(activePresetId);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Не удалось импортировать пресет';
      toast.error(message);
    } finally {
      setSaving(false);
    }
  };

  const runPreview = async () => {
    if (!previewProductId.trim()) {
      toast.error('Укажите SKU или group_id');
      return;
    }
    setPreviewLoading(true);
    try {
      const res = await strategyApi.previewSimilar(
        previewProductId.trim(),
        previewSection,
        previewLimit,
        doc ?? undefined
      );
      setPreviewResult(res);
    } catch (err: unknown) {
      const message =
        (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error
          ?.message || 'Не удалось выполнить preview';
      toast.error(message.includes('not found') || message.includes('product')
        ? `Preview не удался: ${message}`
        : message);
      setPreviewResult(null);
    } finally {
      setPreviewLoading(false);
    }
  };

  const runVisualPreview = async () => {
    setVisualLoading(true);
    try {
      const res = await strategyApi.visualPreview(previewSection, 10, 12);
      setVisualRows(res.rows || []);
      if (!(res.rows || []).length) {
        toast('Выборка пуста — проверьте фид и секцию', { icon: 'ℹ️' });
      }
    } catch (err: unknown) {
      const code = (err as { response?: { data?: { error?: { code?: string } } } })?.response?.data
        ?.error?.code;
      toast.error(
        code === 'cache_not_ready'
          ? 'Кэш похожих ещё прогревается — повторите после завершения прогрева'
          : 'Не удалось загрузить визуальный preview'
      );
      setVisualRows([]);
    } finally {
      setVisualLoading(false);
    }
  };

  const downloadCategoryAudit = async () => {
    setAuditLoading(true);
    const toastId = toast.loading('Считаем аудит категорий… это может занять до пары минут');
    try {
      await strategyApi.downloadCategoryAudit({
        section: previewSection,
        topN: 30,
        maxAvg: auditMaxAvg,
        minSample: 5,
      });
      toast.success('CSV скачан', { id: toastId });
    } catch {
      toast.error('Не удалось выгрузить аудит категорий', { id: toastId });
    } finally {
      setAuditLoading(false);
    }
  };

  if (loading) {
    return <p className="p-8 text-gray-500 text-sm">Загрузка стратегии…</p>;
  }

  if (loadError || !doc) {
    return (
      <div className="max-w-lg mx-auto p-8 text-center">
        <p className="text-red-600 text-sm mb-4">{loadError || 'Стратегия не загружена'}</p>
        <button type="button" onClick={load} className="btn-primary text-sm">
          Повторить
        </button>
      </div>
    );
  }

  const tabs: { id: TabId; label: string; step: string }[] = [
    { id: 'signals', label: 'Сигналы', step: '1' },
    { id: 'selection', label: 'Отбор', step: '2' },
    { id: 'showcase', label: 'Витрина', step: '3' },
    { id: 'preview', label: 'Проверка', step: '4' },
  ];

  const signalSections: { id: SignalSection; label: string }[] = [
    { id: 'weights', label: 'Категория и веса' },
    { id: 'rules', label: 'Правила атрибутов' },
    { id: 'colors', label: 'Цветовые семейства' },
    { id: 'title', label: 'Название' },
    { id: 'synonyms', label: 'Синонимы' },
  ];

  const previewModes: { id: PreviewMode; label: string }[] = [
    { id: 'products', label: 'Проверка по товарам' },
    { id: 'title', label: 'Проверка по названию' },
    { id: 'colors', label: 'Проверка по цветам' },
    { id: 'visual', label: 'Визуал' },
  ];

  return (
    <div className="min-h-screen bg-slate-50">
      <header className="bg-white border-b border-slate-200">
        <div className="max-w-6xl mx-auto px-4 py-4 flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <Link to="/" className="p-2 text-slate-400 hover:text-slate-600 rounded-lg">
              <ArrowLeftIcon className="w-5 h-5" />
            </Link>
            <div>
              <h1 className="text-xl font-bold text-slate-900 flex items-center gap-2">
                <SparklesIcon className="w-6 h-6 text-violet-600" />
                Похожие товары
              </h1>
              <Link to="/outfit-strategy" className="mt-1 inline-flex items-center gap-1 text-xs font-medium text-violet-600 hover:text-violet-800">
                <SquaresPlusIcon className="h-4 w-4" />
                Перейти к алгоритму «Комплекты»
              </Link>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <select
              value={activePresetId}
              onChange={(e) => handleSelectPreset(e.target.value)}
              disabled={saving || !presets.length}
              className="rounded-lg border border-slate-300 text-sm px-2 py-2 min-w-[160px]"
            >
              {presets.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                  {p.is_production ? ' · прод' : ''}
                </option>
              ))}
            </select>
            <input
              type="text"
              value={presetName}
              onChange={(e) => setPresetName(e.target.value)}
              placeholder="Имя пресета"
              className="rounded-lg border border-slate-300 text-sm px-2 py-2 w-40"
            />
            <button
              type="button"
              onClick={handleLoadDefault}
              disabled={saving || !builtinDefault}
              className="btn-outline text-sm"
            >
              По умолчанию
            </button>
            <button
              type="button"
              onClick={handleCreatePreset}
              disabled={saving}
              className="btn-outline text-sm"
              title="Создать новый пресет из текущих настроек"
            >
              <PlusIcon className="w-4 h-4 mr-1 inline" />
              Новый
            </button>
            <button
              type="button"
              onClick={handleExportPreset}
              disabled={saving || !doc}
              className="btn-outline text-sm"
              title="Скачать JSON текущего пресета"
            >
              <ArrowDownTrayIcon className="w-4 h-4 mr-1 inline" />
              Экспорт
            </button>
            <button
              type="button"
              onClick={handleImportPresetClick}
              disabled={saving}
              className="btn-outline text-sm"
              title="Загрузить JSON пресета (новый или замена текущего)"
            >
              <ArrowUpTrayIcon className="w-4 h-4 mr-1 inline" />
              Импорт
            </button>
            <input
              ref={importInputRef}
              type="file"
              accept="application/json,.json"
              className="hidden"
              onChange={handleImportPresetFile}
            />
            <button
              type="button"
              onClick={handleDeletePreset}
              disabled={saving || activePresetId === productionId}
              className="btn-outline text-sm text-red-700 border-red-200 hover:bg-red-50"
            >
              <TrashIcon className="w-4 h-4" />
            </button>
            <label
              className="flex items-center gap-1.5 rounded-lg border border-slate-200 bg-slate-50 px-2 py-2 text-xs text-slate-700 cursor-pointer"
              title="Если выключено — публичный API использует встроенные дефолты кода"
            >
              <input
                type="checkbox"
                checked={doc.enabled}
                onChange={(e) => setDoc({ ...doc, enabled: e.target.checked })}
                className="rounded border-slate-300 text-violet-600"
              />
              Включена
            </label>
            <input
              type="text"
              value={saveReason}
              onChange={(e) => setSaveReason(e.target.value)}
              placeholder="Комментарий к сохранению"
              className="rounded-lg border border-slate-300 text-sm px-2 py-2 w-44"
            />
            <button
              type="button"
              onClick={handleSave}
              disabled={saving || !activePresetId}
              className="btn-outline text-sm"
            >
              <CheckIcon className="w-4 h-4 mr-1 inline" />
              {saving ? '…' : 'Сохранить'}
            </button>
            <button
              type="button"
              onClick={handlePromote}
              disabled={saving || !activePresetId || activePresetId === productionId}
              className="btn-primary text-sm"
            >
              <StarIcon className="w-4 h-4 mr-1 inline" />
              Прогреть кэш
            </button>
          </div>
        </div>
      </header>

      <div className="max-w-6xl mx-auto px-4 py-6">
        <div className="space-y-4">
          {activePresetId && activePresetId !== productionId && (
            <div className="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
              Редактируете пресет «{presetName}». Публичный API и витрина используют продакшн-пресет
              {productionId
                ? ` «${presets.find((p) => p.id === productionId)?.name ?? ''}»`
                : ''}{' '}
              — нажмите «Прогреть кэш», чтобы опубликовать изменения (применятся после прогрева, без простоя).
            </div>
          )}
          {activePresetId && activePresetId === productionId && (
            <div className="rounded-xl border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-900">
              Это продакшн-пресет — сохранённые изменения сразу влияют на публичный API.
            </div>
          )}

          <div className="sticky top-0 z-20 -mx-4 space-y-2 bg-slate-50/95 px-4 py-2 backdrop-blur supports-[backdrop-filter]:bg-slate-50/80">
            <div className="flex flex-wrap gap-1 bg-white rounded-xl border border-slate-200 p-1 shadow-sm">
              {tabs.map((t, i) => (
                <React.Fragment key={t.id}>
                  <button
                    type="button"
                    onClick={() => setTab(t.id)}
                    className={`flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                      tab === t.id
                        ? 'bg-violet-600 text-white'
                        : 'text-slate-600 hover:bg-slate-100'
                    }`}
                  >
                    <span
                      className={`flex h-5 w-5 items-center justify-center rounded-full text-[11px] ${
                        tab === t.id ? 'bg-white/25' : 'bg-slate-100 text-slate-500'
                      }`}
                    >
                      {t.step}
                    </span>
                    {t.label}
                  </button>
                  {i < tabs.length - 1 && (
                    <span className="self-center text-slate-300">›</span>
                  )}
                </React.Fragment>
              ))}
            </div>

            {tab === 'signals' && (
              <div className="flex flex-wrap gap-1 rounded-xl border border-slate-200 bg-white p-1 shadow-sm">
                {signalSections.map((s) => (
                  <button
                    key={s.id}
                    type="button"
                    onClick={() => setSignalSection(s.id)}
                    className={`rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
                      signalSection === s.id
                        ? 'bg-violet-600 text-white'
                        : 'text-slate-600 hover:bg-slate-100'
                    }`}
                  >
                    {s.label}
                  </button>
                ))}
              </div>
            )}

            {tab === 'preview' && (
              <div className="flex flex-wrap gap-1 rounded-xl border border-slate-200 bg-white p-1 shadow-sm">
                {previewModes.map((m) => (
                  <button
                    key={m.id}
                    type="button"
                    onClick={() => setPreviewMode(m.id)}
                    className={`rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
                      previewMode === m.id
                        ? 'bg-violet-600 text-white'
                        : 'text-slate-600 hover:bg-slate-100'
                    }`}
                  >
                    {m.label}
                  </button>
                ))}
              </div>
            )}
          </div>

          <div hidden={tab !== 'showcase'}>
            <RecommendationCompositionPanel />
          </div>

          <div
            hidden={tab === 'showcase'}
            className="bg-white rounded-xl border border-slate-200 shadow-sm p-6"
          >
            {tab === 'signals' && (
              <div className="space-y-4">
                {signalSection === 'weights' && (
                  <SimilarStrategyWeightsTab doc={doc} onChange={setDoc} />
                )}

                {signalSection === 'rules' && (
                  <SimilarStrategyRulesTab
                    doc={doc}
                    onChange={setDoc}
                    featureCatalog={featureCatalog}
                    catalogLoading={catalogLoading}
                  />
                )}

                {signalSection === 'colors' && (
                  <SimilarStrategyColorsTab doc={doc} onChange={setDoc} />
                )}

                {signalSection === 'title' && (
                  <TitleSettings doc={doc} setDoc={setDoc} />
                )}

                {signalSection === 'synonyms' && (
                  <SimilarStrategySynonymsTab doc={doc} onChange={setDoc} />
                )}
              </div>
            )}

            {tab === 'selection' && (
              <div className="space-y-6">
                <ThresholdCascadeExplainer doc={doc} />
                <ThresholdBlock
                  title="Strict"
                  score={doc.thresholds.strict.score_min}
                  base={doc.thresholds.strict.base_without_title_min}
                  strategy={doc.thresholds.strict.strategy}
                  onScore={(v) =>
                    setDoc({
                      ...doc,
                      thresholds: {
                        ...doc.thresholds,
                        strict: { ...doc.thresholds.strict, score_min: v },
                      },
                    })
                  }
                  onBase={(v) =>
                    setDoc({
                      ...doc,
                      thresholds: {
                        ...doc.thresholds,
                        strict: {
                          ...doc.thresholds.strict,
                          base_without_title_min: v,
                        },
                      },
                    })
                  }
                />
                <ThresholdBlock
                  title="Parent fallback"
                  score={doc.thresholds.parent.score_min}
                  base={doc.thresholds.parent.base_without_title_min}
                  strategy={doc.thresholds.parent.strategy}
                  onScore={(v) =>
                    setDoc({
                      ...doc,
                      thresholds: {
                        ...doc.thresholds,
                        parent: { ...doc.thresholds.parent, score_min: v },
                      },
                    })
                  }
                  onBase={(v) =>
                    setDoc({
                      ...doc,
                      thresholds: {
                        ...doc.thresholds,
                        parent: {
                          ...doc.thresholds.parent,
                          base_without_title_min: v,
                        },
                      },
                    })
                  }
                />
                <Field label="Branch score min">
                  <NumberInput
                    value={doc.thresholds.branch.score_min}
                    min={0}
                    max={100}
                    onChange={(v) =>
                      setDoc({
                        ...doc,
                        thresholds: {
                          ...doc.thresholds,
                          branch: { ...doc.thresholds.branch, score_min: v },
                        },
                      })
                    }
                  />
                </Field>
              </div>
            )}

            {tab === 'preview' && (
              <div className="space-y-4">
                <div className="rounded-xl border border-amber-200 bg-amber-50/70 px-4 py-3">
                  <div className="flex flex-wrap items-end gap-3">
                    <div className="min-w-[200px] flex-1">
                      <p className="text-sm font-semibold text-amber-950">Аудит категорий (CSV)</p>
                      <p className="mt-0.5 text-xs text-amber-900/80">
                        Для каждого товара — средний балл категории по топ‑30 кандидатам (каскад
                        leaf/parent/branch). Низкий балл — проверить leaf в PIM. Секция берётся из
                        формы проверки ниже. Секция «Все» — ~2–3 мин, одна секция — быстрее.
                      </p>
                    </div>
                    <div>
                      <label className="mb-1 block text-xs font-medium text-amber-900">
                        Макс. ср. балл
                      </label>
                      <input
                        type="number"
                        min={0}
                        max={200}
                        step={1}
                        value={auditMaxAvg}
                        onChange={(e) => {
                          const v = e.target.value;
                          setAuditMaxAvg(v === '' ? '' : Number(v));
                        }}
                        placeholder="все"
                        className="w-24 rounded-lg border border-amber-300 bg-white px-2 py-1.5 text-sm"
                        title="Пусто — выгрузить все; 70 — только слабые"
                      />
                    </div>
                    <button
                      type="button"
                      onClick={downloadCategoryAudit}
                      disabled={auditLoading}
                      className="btn-outline text-sm border-amber-300 text-amber-950 hover:bg-amber-100"
                    >
                      <ArrowDownTrayIcon className="mr-1 inline h-4 w-4" />
                      {auditLoading ? 'Считаем…' : 'Скачать CSV'}
                    </button>
                  </div>
                </div>

                {previewMode === 'products' && (
                  <SimilarStrategyPreviewTab
                    section={previewSection}
                    productId={previewProductId}
                    limit={previewLimit}
                    loading={previewLoading}
                    result={previewResult}
                    attributeRules={doc?.attribute_rules}
                    colorFamilies={doc?.color_family_dictionary}
                    titleConfig={doc?.title}
                    onSectionChange={setPreviewSection}
                    onProductIdChange={setPreviewProductId}
                    onLimitChange={setPreviewLimit}
                    onRun={runPreview}
                  />
                )}

                {previewMode === 'title' && doc && (
                  <SimilarStrategyTitlePreviewTab
                    section={previewSection}
                    productId={previewProductId}
                    limit={previewLimit}
                    loading={previewLoading}
                    result={previewResult}
                    titleConfig={doc.title}
                    onSectionChange={setPreviewSection}
                    onProductIdChange={setPreviewProductId}
                    onLimitChange={setPreviewLimit}
                    onRun={runPreview}
                  />
                )}

                {previewMode === 'colors' && doc && (
                  <SimilarStrategyColorPreviewTab
                    section={previewSection}
                    productId={previewProductId}
                    limit={previewLimit}
                    loading={previewLoading}
                    result={previewResult}
                    doc={doc}
                    onSectionChange={setPreviewSection}
                    onProductIdChange={setPreviewProductId}
                    onLimitChange={setPreviewLimit}
                    onRun={runPreview}
                  />
                )}

                {previewMode === 'visual' && (
                  <SimilarStrategyVisualTab
                    section={previewSection}
                    loading={visualLoading}
                    rows={visualRows}
                    onSectionChange={setPreviewSection}
                    onRefresh={runVisualPreview}
                  />
                )}
              </div>
            )}

          </div>
        </div>
      </div>
    </div>
  );
};

const Field: React.FC<{ label: string; children: React.ReactNode }> = ({
  label,
  children,
}) => (
  <div>
    <label className="block text-xs font-medium text-slate-600 mb-1">{label}</label>
    {children}
  </div>
);

const TitleSettings: React.FC<{
  doc: SimilarStrategyDocument;
  setDoc: (d: SimilarStrategyDocument) => void;
}> = ({ doc, setDoc }) => (
  <div className="space-y-4">
    <SimilarStrategyTitlePlayground config={doc.title} />
    <label className="flex items-center gap-2 text-sm">
      <input
        type="checkbox"
        checked={doc.title.enabled}
        onChange={(e) => setDoc({ ...doc, title: { ...doc.title, enabled: e.target.checked } })}
      />
      Title matching включён
    </label>
    <label className="flex items-start gap-2 text-sm">
      <input
        type="checkbox"
        className="mt-0.5"
        checked={!doc.title.ignore_category_mismatch}
        onChange={(e) =>
          setDoc({
            ...doc,
            title: {
              ...doc.title,
              ignore_category_mismatch: !e.target.checked,
            },
          })
        }
      />
      <span>
        Снижать баллы названия при слабом совпадении категории
        <span className="block text-xs text-slate-500 mt-0.5">
          Если включено, feature-баллы ограничиваются типом совпадения категории, а при полном
          несовпадении применяется «Cap без категории». Если выключено, название оценивается
          независимо от категории.
        </span>
      </span>
    </label>
    <label className="flex items-start gap-2 text-sm">
      <input
        type="checkbox"
        className="mt-0.5"
        checked={doc.title.ignore_modifier_words_enabled}
        onChange={(e) =>
          setDoc({
            ...doc,
            title: { ...doc.title, ignore_modifier_words_enabled: e.target.checked },
          })
        }
      />
      <span>
        Не штрафовать за лишние слова в названии кандидата
        <span className="block text-xs text-slate-500 mt-0.5">
          Считаем пересечение токенов относительно базового товара: любые дополнительные слова
          только в названии кандидата (тонкого, лёгкого, укороченная и т.д.) не снижают score.
          Слова, которых нет у кандидата, но есть у базы — по-прежнему учитываются.
        </span>
      </span>
    </label>
    <div className="grid sm:grid-cols-3 gap-3">
      {(
        [
          ['exact_score', 'Exact'],
          ['jaccard_max_score', 'Jaccard max'],
          ['model_token_score', 'Model / слово'],
          ['model_max_score', 'Model max'],
          ['feature_token_score', 'Feature / слово'],
          ['feature_max_score', 'Feature max'],
          ['no_category_cap', 'Cap без категории'],
          ['min_token_length', 'Min token len'],
        ] as const
      ).map(([key, label]) => (
        <Field key={key} label={label}>
          <NumberInput
            value={doc.title[key]}
            min={0}
            max={200}
            onChange={(v) => setDoc({ ...doc, title: { ...doc.title, [key]: v } })}
          />
        </Field>
      ))}
    </div>
    <Field label="Stopwords (по одному на строку)">
      <textarea
        rows={4}
        className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm font-mono"
        value={doc.title.stopwords.join('\n')}
        onChange={(e) =>
          setDoc({
            ...doc,
            title: {
              ...doc.title,
              stopwords: e.target.value
                .split('\n')
                .map((s) => s.trim())
                .filter(Boolean),
            },
          })
        }
      />
    </Field>
    <Field label="Model words (посадка, силуэт)">
      <textarea
        rows={6}
        className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm font-mono"
        value={doc.title.model_words.join('\n')}
        onChange={(e) =>
          setDoc({
            ...doc,
            title: {
              ...doc.title,
              model_words: e.target.value
                .split('\n')
                .map((s) => s.trim())
                .filter(Boolean),
            },
          })
        }
      />
    </Field>
    <Field label="Feature words (особенности: принт, лампасы, завязки…)">
      <textarea
        rows={6}
        className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm font-mono"
        value={doc.title.feature_words.join('\n')}
        onChange={(e) =>
          setDoc({
            ...doc,
            title: {
              ...doc.title,
              feature_words: e.target.value
                .split('\n')
                .map((s) => s.trim())
                .filter(Boolean),
            },
          })
        }
      />
      <p className="mt-1 text-xs text-slate-500">
        Отдельный бонус за совпадение токена у базы и кандидата. Добавляйте словоформы явно
        (принт, принтом, лампасы, лампасами).
      </p>
    </Field>
  </div>
);

const ThresholdBlock: React.FC<{
  title: string;
  score: number;
  base: number;
  strategy: string;
  onScore: (v: number) => void;
  onBase: (v: number) => void;
}> = ({ title, score, base, strategy, onScore, onBase }) => (
  <div className="border border-slate-100 rounded-lg p-4">
    <p className="text-sm font-semibold text-slate-800 mb-2">{title}</p>
    <p className="mb-3 text-xs text-slate-600">
      <span className="text-slate-800">{strategyStageLabel(strategy)}</span>
      <span className="ml-1.5 font-mono text-slate-400">{strategy}</span>
    </p>
    <div className="grid grid-cols-2 gap-3">
      <Field label="score_min">
        <NumberInput value={score} min={0} max={200} onChange={onScore} />
      </Field>
      <Field label="base_without_title_min">
        <NumberInput value={base} min={0} max={200} onChange={onBase} />
      </Field>
    </div>
  </div>
);
