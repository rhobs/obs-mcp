import { AbsoluteTimeRange, RelativeTimeRange, TimeRangeValue } from '@perses-dev/core';
import { Panel } from '@perses-dev/dashboards';
import {
  DataQueriesProvider,
  TimeRangeProvider,
  useSuggestedStepMs,
} from '@perses-dev/plugin-system';
import { DEFAULT_PROM } from '@perses-dev/prometheus-plugin';
import React, { useMemo } from 'react';
import useResizeObserver from 'use-resize-observer';
import PersesWidgetWrapper from '../PersesWidgetWrapper';

type PersesTableProps = {
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

const TimeSeries = ({ query }: PersesTableProps) => {
  const datasource = DEFAULT_PROM;
  const { width, ref: boxRef } = useResizeObserver();
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
    <div ref={boxRef} style={{ width: '100%', height: '100%', overflow: 'auto' }}>
      <DataQueriesProvider definitions={definitions} options={{ suggestedStepMs, mode: 'range' }}>
        <Panel
          panelOptions={{
            hideHeader: false,
          }}
          definition={{
            kind: 'Panel',
            spec: {
              queries: [],
              display: { name: '' },
              plugin: {
                kind: 'Table',
                spec: {
                  visual: {
                    stack: 'all',
                  },
                },
              },
            },
          }}
        />
      </DataQueriesProvider>
    </div>
  );
};

const PersesTable = (props: PersesTableProps) => {
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

export default PersesTable;
