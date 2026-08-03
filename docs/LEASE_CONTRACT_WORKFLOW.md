# Contratos de arriendo versionados

El módulo de contratación asistida vive dentro del expediente de cada propiedad, en la sección **Contratación**.

## Flujo

1. Crear una postulación con identidad, contacto, ocupantes e información laboral del interesado.
2. Cargar los antecedentes entregados por el postulante. El informe comercial/DICOM se guarda como documento restringido; la plataforma no consulta servicios externos ni toma decisiones automáticas.
3. Definir las condiciones económicas, garantía, reajuste, servicios y garantía adicional.
4. Generar una versión inmutable del borrador. El sistema produce un PDF para revisión visual y un DOCX editable para la revisión jurídica y las modificaciones de la notaría.
5. Registrar quién revisó el texto. Sólo una versión aprobada puede recibir el contrato formalizado.
6. Imprimir o editar el DOCX, firmar y formalizar fuera de la plataforma.
7. Cargar el contrato firmado/notariado como PDF y confirmar la revisión de identidad, firmas, certificación, páginas y anexos.

Los archivos fuente y el documento firmado se conservan por separado. Una modificación notarial relevante debe originar una nueva versión o quedar reflejada en el PDF firmado; nunca se sobrescribe el borrador original.

## Garantías

El formulario separa expresamente el primer mes de renta del depósito en garantía. La versión inicial admite de uno a tres meses, pero los valores superiores al estándar deben someterse a revisión jurídica individual.

Las garantías adicionales soportadas son codeudor solidario, seguro sujeto a aceptación y póliza, o ninguna garantía adicional.

La generación automática de un cheque en garantía está bloqueada. Debido a su naturaleza de orden de pago a la vista, una cláusula de ese tipo requiere preparación jurídica específica.

## Seguridad documental

- Los antecedentes comerciales reciben confidencialidad `restricted`.
- La carga exige confirmar que el documento fue entregado por el postulante o se obtuvo con autorización.
- Cada asociación, generación, revisión y formalización produce un evento de auditoría.
- Cada PDF y DOCX conserva su hash SHA-256, versión de plantilla y versión de base jurídica.
- El contrato firmado sólo puede asociarse a la misma organización y propiedad.

## Migración

Aplicar `migrations/00015_lease_contract_workflow.sql` antes de desplegar la API y la web actualizadas.

La plantilla `cl-residential-v1.0` es un borrador asistido. Debe ser revisada por un abogado chileno antes de utilizarse como plantilla jurídica aprobada para producción.
