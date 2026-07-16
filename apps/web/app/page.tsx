const foundations = [
  "Separación de datos por organización",
  "PostgreSQL como fuente oficial",
  "Archivos privados en almacenamiento de objetos",
  "Aprobación humana para acciones sensibles",
];

export default function Home() {
  return (
    <main>
      <section className="hero">
        <p className="eyebrow">Fundación del producto</p>
        <h1>Control Propiedades</h1>
        <p className="summary">
          Una plataforma para organizar la operación, las finanzas y el expediente documental de cada propiedad.
        </p>
        <div className="status" role="status">
          <span aria-hidden="true" />
          Base técnica preparada
        </div>
      </section>

      <section className="principles" aria-labelledby="principles-title">
        <h2 id="principles-title">Principios desde el inicio</h2>
        <ul>
          {foundations.map((foundation) => (
            <li key={foundation}>{foundation}</li>
          ))}
        </ul>
      </section>
    </main>
  );
}
