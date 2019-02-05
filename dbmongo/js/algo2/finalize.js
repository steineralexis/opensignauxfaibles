function finalize(k, v) {


  //var empty = (v.entreprise||[]).reduce((accu, siren_periode) => {
  //  if (siren_periode){
  //    Object.keys(siren_periode).forEach(key => {
  //      accu[key] = null
  //    })
  //  }
  //  return(accu)
  //},{})
  //
  ///
  ///////////////////////////////////////////////
  // consolidation a l'echelle de l'entreprise //
  ///////////////////////////////////////////////
  ///
  //

  let etablissements_connus = []
  let entreprise = (v.entreprise || {})

  Object.keys(v).forEach(siret =>{
    if (siret != "entreprise" && siret != "siren" ) {
      etablissements_connus[siret] = true
      //if (v[siret]){  // always TRUE
      //    var time = v[siret].periode.getTime()
      entreprise.effectif_entreprise = (entreprise.effectif_entreprise || 0) + v[siret].effectif // initialized to null
      entreprise.apart_entreprise = (entreprise.apart_entreprise || 0) + v[siret].apart_heures_consommees // initialized to 0
      entreprise.debit_entreprise = (entreprise.debit_entreprise || 0) +
        (v[siret].montant_part_patronale || 0) +
        (v[siret].montant_part_ouvriere || 0) 
      // not initialized
      //}
    }
  })

  Object.keys(v).forEach(siret =>{ 
      Object.assign(v[siret], entreprise) 
  })

  //
  ///
  //////////////////////////////
  /// Objectif entrainement ///
  //////////////////////////////
  ///
  //

  Object.keys(v).forEach(siret => {
    if (siret != "entreprise" && siret != "siren" && v[siret]){

      if ("time_til_outcome" in v[siret] && 
        v[siret].time_til_outcome <= 18){
        //||
        //(("arrete_bilan_diane" in v[siret] || "arrete_bilan_bdf" in v[siret]) && 
        //  v[siret].time_til_outcome <= 30)) &&
        //  !("arrete_bilan_diane" in v[siret] && v[siret].arrete_bilan_diane < key.periode &&  
        //  generatePeriodSerie(key.periode, v[siret].arrete_bilan_diane).length >= 18)  &&
        //!("arrete_bilan_bdf" in v[siret] && v[siret].arrete_bilan_bdf < key.periode && 
        //  generatePeriodSerie(key.periode, v[siret].arrete_bilan_bdf).length >= 18)) {
        v[siret].outcome = true
      } else 
        v[siret].outcome = false
    }
  })

  //une fois que les comptes sont faits...
  let output = []
  Object.keys(v).forEach(siret =>{
    if (siret != "entreprise" && siret != "siren" && v[siret]) {
      v[siret].nbr_etablissements_connus = Object.keys(etablissements_connus).length
      output.push(v[siret])
    }
  })

  if (output.length > 0)
    return output
}
