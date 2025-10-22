import { AbsoluteTimeRange, RelativeTimeRange, TimeRangeValue } from '@perses-dev/core';
import { Panel } from '@perses-dev/dashboards';
import {
  DataQueriesProvider,
  TimeRangeProvider,
  useSuggestedStepMs,
} from '@perses-dev/plugin-system';
import { DEFAULT_PROM } from '@perses-dev/prometheus-plugin';
import React, { useMemo, useRef } from 'react';
import useResizeObserver from 'use-resize-observer';
import PersesWidgetWrapper from '../PersesWidgetWrapper';

type PersesPieChartProps = {
  duration: string;
  end: string;
  query: string;
  start: string;
  step: string;
};

const useTimeRange = (start?: string, end?: string, duration?: string) => {
  const result = useMemo(() => {
    let timeRange: TimeRangeValue;
    if (start && end) {
      timeRange = {
        start: new Date(start),
        end: new Date(end),
      } as AbsoluteTimeRange;
    } else {
      timeRange = { pastDuration: duration || '1h' } as RelativeTimeRange;
    }
    return timeRange;
  }, [duration, end, start]);
  return result;
};

const TimeSeries = ({ query }: PersesPieChartProps) => {
  const datasource = DEFAULT_PROM;
  const panelRef = useRef<HTMLDivElement>(null);
  const { width } = useResizeObserver({ ref: panelRef });
  const suggestedStepMs = useSuggestedStepMs(width);

  const definitions =
    query !== ''
      ? [
          {
            kind: 'PrometheusTimeSeriesQuery',
            spec: {
              datasource: {
                kind: datasource.kind,
                name: datasource.name,
              },
              query: query,
            },
          },
        ]
      : [];

  return (
    <div ref={panelRef} style={{ width: '100%', height: '100%' }}>
      <DataQueriesProvider definitions={definitions} options={{ suggestedStepMs, mode: 'range' }}>
        <Panel
          panelOptions={{
            hideHeader: true,
          }}
          definition={{
            kind: 'Panel',
            spec: {
              queries: [],
              display: { name: '' },
              plugin: {
                kind: 'PieChart',
                spec: {
                  calculation: 'last',
                  legend: { placement: 'right' },
                  value: { placement: 'center' },
                },
              },
            },
          }}
        />
      </DataQueriesProvider>
    </div>
  );
};

const PersesPieChart = (props: PersesPieChartProps) => {
  const timeSeriesProps = props;
  const timeRange = useTimeRange(
    timeSeriesProps.start,
    timeSeriesProps.end,
    timeSeriesProps.duration,
  );
  return (
    <PersesWidgetWrapper>
      <TimeRangeProvider timeRange={timeRange} refreshInterval="0s">
        <TimeSeries {...timeSeriesProps} />
      </TimeRangeProvider>
    </PersesWidgetWrapper>
  );
};

export default PersesPieChart;
