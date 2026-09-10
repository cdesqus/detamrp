import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { MaterialRequirements, RequirementResult } from './material-requirements';

const result: RequirementResult = {
 nodes: [
  { path:'0',parentPath:'',itemId:'fg',kind:'FG',itemCode:'FG-A',name:'Assembly',unit:'PCS',usage:'1',quantity:'10',terminal:false,depth:0 },
  { path:'0.0',parentPath:'0',itemId:'x',kind:'RAW_MATERIAL',itemCode:'X',name:'Bracket',unit:'PCS',usage:'2',quantity:'20',terminal:false,depth:1 },
  { path:'0.0.0',parentPath:'0.0',itemId:'plate',kind:'RAW_MATERIAL',itemCode:'001',name:'Plate',unit:'KG',usage:'0.3',quantity:'6',terminal:true,depth:2 },
 ],
 materials:[{itemId:'plate',itemCode:'001',name:'Plate',unit:'KG',quantity:'6',unitPrice:'20',currency:'IDR',value:'120',qtyPerKanban:'50',kanbanEquivalent:'0.12',purchaseKanban:'1',paths:['0.0.0']}],
 totals:{IDR:'120'},costComplete:true,
};
describe('MaterialRequirements',()=>{
 it('expands nested components without conflating per-parent usage and total requirements',()=>{
  render(<MaterialRequirements result={result} showCosts />);
  expect(screen.queryByRole('button',{name:'Collapse FG-A'})).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button',{name:'Expand FG-A'}));
  expect(screen.getByRole('button',{name:'Expand X'})).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button',{name:'Expand all'}));
  expect(screen.getByRole('button',{name:'Collapse X'})).toBeInTheDocument();
  expect(screen.getByText('Usage / Parent Unit')).toBeInTheDocument();
  expect(screen.getByText('Quantity Required')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button',{name:'Collapse all'}));
  expect(screen.queryByRole('button',{name:'Expand X'})).not.toBeInTheDocument();
 });
 it('omits valuation for users without cost permission',()=>{
  render(<MaterialRequirements result={result} showCosts={false} />);
  expect(screen.queryByText('Unit Price')).not.toBeInTheDocument();
  expect(screen.queryByText('Material Value')).not.toBeInTheDocument();
  expect(screen.getByText('001')).toBeInTheDocument();
 });
});
