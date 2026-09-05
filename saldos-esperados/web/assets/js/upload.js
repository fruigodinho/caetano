// caetano / saldos-esperados — parsing e envio em lote de balancetes CSV.
// Partilhado entre o upload de ficheiro (data.html) e a colagem manual
// (data_manual.html): ambos produzem {label, entries} e enviam pela mesma
// API em três passos (init/batch/finish).
(function () {
  "use strict";

  /**
   * Interpreta o conteúdo de um balancete CSV (separador ';', 1ª linha =
   * rótulo, 4 linhas de cabeçalho, contas com 8 dígitos, saldo em formato
   * europeu "1.234,56").
   * @param {string} content
   * @returns {{label: string, entries: Array<{account_number:string, account_name:string, balance:number}>}}
   */
  function parseCSVContent(content) {
    const lines = content
      .split("\n")
      .map((l) => l.trim())
      .filter((l) => l.length > 0);

    if (lines.length < 5) {
      throw new Error("Conteúdo CSV inválido (muito poucas linhas)");
    }

    const label = lines[0];
    const dataLines = lines.slice(4);
    const entries = [];

    for (const line of dataLines) {
      const cols = line.split(";").map((c) => c.trim());
      if (cols.length < 3) continue;

      const accountNumber = cols[0];
      const accountName = cols[1];
      let balanceStr = cols[2];

      if (accountNumber.length !== 8) continue;

      balanceStr = balanceStr.replace(/\./g, "").replace(",", ".");
      const balance = parseFloat(balanceStr);
      if (isNaN(balance)) continue;

      entries.push({ account_number: accountNumber, account_name: accountName, balance: balance });
    }

    if (entries.length === 0) {
      throw new Error("Nenhuma conta válida encontrada no conteúdo");
    }

    return { label, entries };
  }

  /**
   * Envia as entradas em lotes de 100 via /saldos-esperados/data/{init,batch,finish}.
   * @param {string} filename
   * @param {string} label
   * @param {Array} entries
   * @param {(percent:number, status:string)=>void} onProgress
   */
  async function uploadEntries(filename, label, entries, onProgress) {
    const batchSize = 100;

    onProgress(5, "A iniciar...");
    const initResp = await fetch("/saldos-esperados/data/init", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ filename, label }),
    });
    if (!initResp.ok) {
      throw new Error((await initResp.json()).error || "Erro ao iniciar upload");
    }
    const { upload_id: uploadID } = await initResp.json();

    const totalBatches = Math.ceil(entries.length / batchSize);
    for (let i = 0; i < totalBatches; i++) {
      const batch = entries.slice(i * batchSize, (i + 1) * batchSize);
      onProgress(5 + ((i + 1) / totalBatches) * 85, "A processar...");

      const batchResp = await fetch("/saldos-esperados/data/batch", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ upload_id: uploadID, entries: batch }),
      });
      if (!batchResp.ok) {
        throw new Error((await batchResp.json()).error || "Erro ao processar lote");
      }
    }

    onProgress(95, "A finalizar...");
    const finishResp = await fetch("/saldos-esperados/data/finish", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ upload_id: uploadID }),
    });
    if (!finishResp.ok) {
      throw new Error((await finishResp.json()).error || "Erro ao finalizar upload");
    }
    onProgress(100, "Concluído!");
  }

  window.caeUploadCSV = { parseCSVContent, uploadEntries };
})();
