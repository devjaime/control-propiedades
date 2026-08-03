# Estado de implementación

Actualizado el 16 de julio de 2026.

## Arquitectura actual

- Web con Next.js App Router y TypeScript.
- API HTTP en Go con Chi y PostgreSQL mediante pgx.
- PostgreSQL como fuente oficial del estado operacional.
- MinIO para almacenamiento privado de PDF e imágenes.
- Migraciones versionadas con Goose.
- Docker Compose para PostgreSQL y MinIO en desarrollo local.
- Separación de datos por organización y auditoría para acciones relevantes.

## Funcionalidades terminadas

### Acceso y propiedades

- Registro de usuario y organización.
- Inicio y cierre de sesión mediante cookie HTTP-only.
- Registro y consulta de propiedades aisladas por organización.
- Navegación adaptable a escritorio y teléfono.

### Documentos

- Carga privada de PDF, JPEG, PNG y WebP de hasta 50 MB.
- Almacenamiento en MinIO y metadatos estructurados en PostgreSQL.
- Hash SHA-256, detección de duplicados, etiquetas y búsqueda.
- Clasificación, revisión humana, rechazo y verificación inmutable.
- Descarga autenticada de documentos privados.

### Contratos y arrendatarios

- Un arrendatario queda asociado de forma inmutable durante cada vigencia contractual.
- Para cambiar de arrendatario se cierra el contrato con fecha y motivo auditados y se crea uno nuevo.
- Inicio, término, renta base, condiciones y calendario mensual de pagos.
- Entre una y doce cuotas mensuales con día y monto esperado.
- Separación explícita entre la obligación contractual original y un acuerdo especial de pago en cuotas.
- Correcciones mediante enmiendas auditadas, sin modificar el arrendatario ni períodos que ya tienen pagos.
- Aviso configurable para renovación y anticipación en meses y días para solicitar la restitución.
- Reajuste por IPC programado como alerta; nunca cambia la renta sin revisión humana.

### Cobros, pagos y comprobantes

- Generación automática del cobro mensual durante la vigencia del contrato.
- Cobros opcionales de luz, agua, basura, gastos comunes y otros conceptos.
- Uno o más depósitos hasta completar el arriendo mensual.
- Respaldo documental, fecha, medio, referencia y observaciones por abono.
- Conciliación y confirmación humana obligatoria.
- Clasificación de pagos en fecha o atrasados según la cuota planificada.
- Vale PDF inmutable al completar el arriendo, con folio, firma electrónica simple, detalle de cuotas y QR.
- Página pública de verificación con información personal limitada.
- Snapshot, hash y documento del vale conservados en la base de datos.

### Alertas, calendario y mantenciones

- Notificaciones internas para cuotas, pagos por revisar y saldos vencidos.
- Alertas de renovación contractual, restitución, IPC y mantenciones.
- Mantenciones periódicas con categoría, frecuencia, próxima fecha, anticipación y última ejecución.
- Cálculo automático de la próxima mantención después de marcarla realizada.
- Calendario rodante de tres meses desde el período seleccionado.
- Sugerencias trimestrales de revisión financiera, contractual, documental y de la propiedad.
- Las alertas revisadas permanecen visibles en el calendario para conservar contexto.

## Modelo y migraciones

1. `00001_foundation.sql`: organizaciones, usuarios, propiedades y auditoría.
2. `00002_accounts_and_documents.sql`: sesiones, documentos y almacenamiento privado.
3. `00003_document_review.sql`: metadatos y flujo de revisión documental.
4. `00004_rentals_payments_receipts.sql`: arrendatarios, contratos, cobros, pagos, observaciones, notificaciones y vales.
5. `00005_lease_installment_schedule.sql`: calendario normalizado de cuotas y asignación planificada de pagos.
6. `00006_contract_alerts_and_maintenance.sql`: ciclo contractual, IPC, restitución y mantenciones periódicas.
7. `00007_lease_amendments_and_contract_terms.sql`: enmiendas, obligación contractual, acuerdos especiales e inspección de salida.

## Validaciones realizadas

- Pruebas unitarias de API y generación del PDF.
- ESLint y verificación TypeScript.
- Compilación de producción de API y web.
- Auditoría npm sin vulnerabilidades conocidas al momento de la revisión.
- Pruebas integrales temporales para contratos con dos cuotas, pago tardío, emisión del PDF, QR, alertas contractuales, cierre de contrato, mantenciones y calendario de tres meses.
- Los datos creados para las pruebas integrales fueron eliminados al terminar.

## Próximas iteraciones sugeridas

- Ejecutor programado independiente para actualizar alertas sin depender de abrir la propiedad.
- Canales externos de notificación, como correo o calendario.
- Renovación contractual mediante anexos y aprobación humana explícita.
- Reversión auditada de pagos y reemplazo o anulación de comprobantes.
- Órdenes de trabajo, proveedores, presupuestos y gastos de mantención.
- Adaptadores de solo lectura y propuestas para OpenClaw y HermesAgents.
- Políticas de autorización más granulares por rol y propiedad.

Los agentes futuros podrán analizar los registros y crear propuestas, pero no deberán confirmar pagos, modificar contratos vigentes ni ejecutar acciones financieras sin aprobación humana.
