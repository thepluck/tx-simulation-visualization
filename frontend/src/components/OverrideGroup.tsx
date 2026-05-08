export type OverrideField<Row> = {
  key: keyof Row;
  label: string;
  placeholder?: string;
};

type OverrideGroupProps<Row extends object> = {
  createRow: () => Row;
  fields: OverrideField<Row>[];
  rows: Row[];
  title: string;
  onRowsChange: (rows: Row[]) => void;
};

export default function OverrideGroup<Row extends object>(props: OverrideGroupProps<Row>) {
  const { createRow, fields, rows, title, onRowsChange } = props;

  const addRow = () => onRowsChange([...rows, createRow()]);
  const removeRow = (index: number) => onRowsChange(rows.filter((_, rowIndex) => rowIndex !== index));
  const updateRow = (index: number, key: keyof Row, value: string) => {
    onRowsChange(rows.map((row, rowIndex) => (rowIndex === index ? { ...row, [key]: value } : row)));
  };

  return (
    <section className="override-group">
      <div className="override-group-header">
        <h3>{title}</h3>
        <button type="button" className="add-row-button" title={`Add ${title}`} onClick={addRow}>
          +
        </button>
      </div>
      {rows.length === 0 ? (
        <div className="override-empty">None</div>
      ) : (
        <div className="override-list">
          {rows.map((row, index) => (
            <div className="override-row" key={index}>
              {fields.map((field) => (
                <label key={String(field.key)}>
                  {field.label}
                  <input
                    value={String(row[field.key] ?? "")}
                    placeholder={field.placeholder}
                    onChange={(event) => updateRow(index, field.key, event.target.value)}
                  />
                </label>
              ))}
              <button type="button" className="remove-row-button" title={`Remove ${title}`} onClick={() => removeRow(index)}>
                x
              </button>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}
